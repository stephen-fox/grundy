package main

import (
	"errors"
	"flag"
	"io"
	"log"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/kardianos/service"
	"github.com/stephen-fox/grundy/internal/gcw"
	"github.com/stephen-fox/grundy/internal/settings"
	"github.com/stephen-fox/grundy/internal/steamw"
	"github.com/stephen-fox/watcher"
)

const (
	name        = "grundy"
	description = "Grundy crushes your games into Steam shortcuts " +
		"so you do not have to! Please refer to the usage documentation " +
		"at https://github.com/stephen-fox/grundy for more information."

	daemonCommandArg      = "daemon"
	appSettingsDirPathArg = "settings"
	helpArg               = "h"
)

type primarySettings struct {
	watcher             watcher.Watcher
	watcherConfig       watcher.Config
	app                 settings.AppSettings
	launchers           settings.LaunchersSettings
	knownGames          settings.KnownGamesSettings
	steamShortcutsMutex *sync.Mutex
	lock                settings.Lock
}

var (
	daemonCommand      = flag.String(daemonCommandArg, "", "Manage the application's daemon")
	appSettingsDirPath = flag.String(appSettingsDirPathArg, settings.DirPath(), "The directory to store application settings")
	help               = flag.Bool(helpArg, false, "Show this help information")
)

type application struct {
	primary *primarySettings
	stop    chan chan struct{}
}

func (o *application) Start(s service.Service) error {
	log.Println("Acquiring lock...")

	err := o.primary.lock.Acquire()
	if err != nil {
		return err
	}

	log.Println("Lock acquired")

	go mainLoop(o.primary, o.stop)

	return nil
}

func (o *application) Stop(s service.Service) error {
	log.Println("Stopping...")

	c := make(chan struct{})
	o.stop <- c
	<-c

	log.Println("Finished stopping resources")

	return nil
}

func main() {
	flag.Parse()

	if *help {
		flag.PrintDefaults()
		os.Exit(0)
	}

	logFile, err := settings.LogFile(*appSettingsDirPath)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer logFile.Close()

	log.SetOutput(io.MultiWriter(logFile, os.Stderr))

	primary, err := setupPrimarySettings(*appSettingsDirPath)
	if err != nil {
		log.Fatal(err.Error())
	}

	serviceConfig, err := serviceConfig()
	if err != nil {
		log.Fatal(err.Error())
	}

	app := &application{
		primary: primary,
		stop:    make(chan chan struct{}),
	}

	s, err := service.New(app, serviceConfig)
	if err != nil {
		log.Fatal(err.Error())
	}

	if len(strings.TrimSpace(*daemonCommand)) > 0 {
		*daemonCommand = strings.ToLower(*daemonCommand)
		if *daemonCommand == "status" {
			status, err := s.Status()
			if err != nil {
				log.Fatal(err.Error())
			}

			switch status {
			case service.StatusRunning:
				log.Println("Daemon status - running")
			case service.StatusStopped:
				log.Println("Daemon status - stopped")
			case service.StatusUnknown:
				log.Println("Daemon status - unknown")
			default:
				log.Println("Daemon status could not be determined")
			}
		} else {
			log.Println("Executing command '" + *daemonCommand + "'...")

			err = service.Control(s, *daemonCommand)
			if err != nil {
				log.Fatal(err.Error())
			}
		}

		os.Exit(0)
	}

	err = s.Run()
	if err != nil {
		log.Fatal(err.Error())
	}
}

func setupPrimarySettings(settingsDirPath string) (*primarySettings, error) {
	launchers := settings.NewLaunchersSettings()
	launchers.AddOrUpdate(settings.NewLauncher())
	app := settings.NewAppSettings()
	exampleGame := settings.NewGameSettings()

	saveableToShouldCreateInSettingsDir := map[settings.SaveableSettings]bool{
		launchers:   true,
		app:         true,
		exampleGame: false,
	}

	for s, createInMainDir := range saveableToShouldCreateInSettingsDir {
		err := settings.Create(settingsDirPath + "/examples", settings.ExampleSuffix, s)
		if err != nil {
			return &primarySettings{}, errors.New("Failed to create example application settings file - " + err.Error())
		}

		if createInMainDir {
			_, statErr := os.Stat(path.Join(settingsDirPath, s.Filename("")))
			if statErr != nil {
				err := settings.Create(settingsDirPath, "", s)
				if err != nil {
					return &primarySettings{}, err
				}
			}
		}
	}

	internalDirPath, err := settings.CreateInternalFilesDir(settingsDirPath)
	if err != nil {
		return &primarySettings{}, errors.New("Failed to create internal settings directory path - " + err.Error())
	}

	steamShortcutsMutex := &sync.Mutex{}

	knownGames, loaded := settings.LoadOrCreateKnownGamesSettings(internalDirPath)
	if loaded {
		err := cleanupKnownGameShortcuts(steamShortcutsMutex, knownGames)
		if err != nil {
			log.Println("Failed to cleanup known game shortcuts -", err.Error())
		}
	}

	primarySettingsWatcherConfig := watcher.Config{
		ScanFunc:    watcher.ScanFilesInDirectory,
		RootDirPath: settingsDirPath,
		FileSuffix:  settings.FileExtension,
		Changes:     make(chan watcher.Changes),
	}

	primarySettingsWatcher, err := watcher.NewWatcher(primarySettingsWatcherConfig)
	if err != nil {
		return &primarySettings{}, errors.New("Failed to watch application settings directory for changes - " + err.Error())
	}

	return &primarySettings{
		watcherConfig:       primarySettingsWatcherConfig,
		watcher:             primarySettingsWatcher,
		app:                 app,
		launchers:           launchers,
		knownGames:          knownGames,
		steamShortcutsMutex: steamShortcutsMutex,
		lock:                settings.NewLock(internalDirPath),
	}, nil
}

func cleanupKnownGameShortcuts(fileMutex *sync.Mutex, knownGames settings.KnownGamesSettings) error {
	var targets []string

	m := knownGames.RemoveNonExistingConfigs()

	if len(m) == 0 {
		return nil
	}

	for _, gameName := range m {
		targets = append(targets, gameName)
	}

	info, err := steamw.NewSteamDataInfo()
	if err != nil {
		return err
	}

	config := steamw.DeleteShortcutConfig{
		GameNames:  targets,
		Info:       info,
		FileAccess: fileMutex,
	}

	result := steamw.DeleteShortcutPerId(config)

	for id, deleted := range result.IdsToDeletedGames {
		log.Println("Deleted shortcut for", deleted, "for Steam ID", id)
	}

	for id, notDeleted := range result.IdsToNotDeletedGames {
		log.Println("Deleted shortcut for", notDeleted, "does not exist for Steam ID", id)
	}

	for id, err := range result.IdsToFailures {
		log.Println("Failed to cleanup shortcut for Steam user ID", id, "-", err.Error())
	}

	return nil
}

func mainLoop(primary *primarySettings, stop chan chan struct{}) {
	primary.watcher.Start()
	dirPathsToWatchers  := make(map[string]watcher.Watcher)
	updateBuffer := 5 * time.Second
	watchersTimer := time.NewTimer(updateBuffer)
	stopTimerSafely(watchersTimer)
	defer watchersTimer.Stop()

	for {
		select {
		case change := <- primary.watcherConfig.Changes:
			for _, filePath := range change.UpdatedFilePaths {
				log.Println("Main settings file has been updated:", filePath)

				switch path.Base(filePath) {
				case primary.app.Filename(""):
					err := primary.app.Reload(filePath)
					if err != nil {
						log.Println("Failed to load application settings -", err.Error())
						continue
					}
				case primary.launchers.Filename(""):
					err := primary.launchers.Reload(filePath)
					if err != nil {
						log.Println("Failed to load launchers settings -", err.Error())
						continue
					}
					// TODO: Update shortcuts when this happens.
				default:
					continue
				}

				stopTimerSafely(watchersTimer)

				watchersTimer.Reset(updateBuffer)
			}
		case <-watchersTimer.C:
			updateGameCollectionWatchers(primary, dirPathsToWatchers)
		case err := <-primary.lock.Errs():
			log.Println("Error maintaining lock file - " + err.Error())
		case c := <-stop:
			for k, w := range dirPathsToWatchers {
				w.Destroy()

				delete(dirPathsToWatchers, k)
			}

			primary.watcher.Destroy()

			primary.lock.Release()

			c <- struct{}{}

			return
		}
	}
}

func stopTimerSafely(t *time.Timer) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
}

func updateGameCollectionWatchers(primary *primarySettings, dirPathsToWatchers map[string]watcher.Watcher) {
	log.Println("Updating game collection watchers...")

	watchDirs := primary.app.WatchPaths()

	OUTER:
	for dirPath, currentWatcher := range dirPathsToWatchers {
		for _, newDirPath := range watchDirs {
			if dirPath == newDirPath {
				continue OUTER
			}
		}

		log.Println("No longer watching", dirPath)

		currentWatcher.Destroy()

		delete(dirPathsToWatchers, dirPath)
	}

	for _, dirPath := range watchDirs {
		_, ok := dirPathsToWatchers[dirPath]
		if ok {
			continue
		}

		collectionWatcherConfig := &gcw.WatcherConfig{
			AppSettingsDirPath:  *appSettingsDirPath,
			DirPath:             dirPath,
			Launchers:           primary.launchers,
			KnownGames:          primary.knownGames,
			SteamShortcutsMutex: primary.steamShortcutsMutex,
		}

		w, err := gcw.NewGameCollectionWatcher(collectionWatcherConfig)
		if err != nil {
			log.Println(err.Error())
			continue
		}

		log.Println("Now watching subdirectories in", dirPath)

		w.Start()

		dirPathsToWatchers[dirPath] = w
	}
}
