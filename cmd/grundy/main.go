package main

import (
	"flag"
	"io"
	"log"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/stephen-fox/grundy/internal/gcw"
	"github.com/stephen-fox/grundy/internal/settings"
	"github.com/stephen-fox/grundy/internal/steamw"
	"github.com/stephen-fox/watcher"
	"github.com/stephen-fox/grundy/internal/dman"
)

const (
	daemonCommandArg      = "daemon"
	appSettingsDirPathArg = "settings"
	helpArg               = "h"
)

type primarySettings struct {
	app                 settings.AppSettings
	launchers           settings.LaunchersSettings
	knownGames          settings.KnownGamesSettings
	steamShortcutsMutex *sync.Mutex
	dirPathsToWatchers  map[string]watcher.Watcher
}

var (
	daemonCommand      = flag.String(daemonCommandArg, "", "Manage the application's daemon with one of the following commands:\n" + dman.AvailableManagementCommands())
	appSettingsDirPath = flag.String(appSettingsDirPathArg, settings.DirPath(), "The directory to store application settings")
	help               = flag.Bool(helpArg, false, "Show this help information")
)

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

	if len(strings.TrimSpace(*daemonCommand)) > 0 {
		daemonManager, err := dman.NewManager()
		if err != nil {
			log.Fatal(err.Error())
		}

		status, err := daemonManager.DoManagementCommand(*daemonCommand)
		if err != nil {
			log.Fatal(err.Error())
		}

		log.Println("Daemon status - " + status)

		os.Exit(0)
	}

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
		err := settings.Create(*appSettingsDirPath + "/examples", settings.ExampleSuffix, s)
		if err != nil {
			log.Fatal("Failed to create example application settings files - " + err.Error())
		}

		if createInMainDir {
			_, statErr := os.Stat(path.Join(*appSettingsDirPath, s.Filename("")))
			if statErr != nil {
				err := settings.Create(*appSettingsDirPath, "", s)
				if err != nil {
					log.Fatal(err.Error())
				}
			}
		}
	}

	primarySettingsWatcherConfig := watcher.Config{
		ScanFunc:    watcher.ScanFilesInDirectory,
		RootDirPath: *appSettingsDirPath,
		FileSuffix:  settings.FileExtension,
		Changes:     make(chan watcher.Changes),
	}

	primarySettingsWatcher, err := watcher.NewWatcher(primarySettingsWatcherConfig)
	if err != nil {
		log.Fatal("Failed to watch application settings directory for changes")
	}
	defer primarySettingsWatcher.Stop()

	primarySettingsWatcher.Start()

	steamShortcutsMutex := &sync.Mutex{}

	internalDirPath, err := settings.CreateInternalFilesDir(*appSettingsDirPath)
	if err != nil {
		log.Fatal("Failed to create internal settings directory path - " + err.Error())
	}

	knownGames, loaded := settings.LoadOrCreateKnownGamesSettings(internalDirPath)
	if loaded {
		log.Println("Loaded existing known game settings")

		err := cleanupKnownGameShortcuts(steamShortcutsMutex, knownGames)
		if err != nil {
			log.Println("Failed to cleanup known game shortcuts -", err.Error())
		}
	}

	primary := &primarySettings{
		app:                 app,
		launchers:           launchers,
		knownGames:          knownGames,
		steamShortcutsMutex: steamShortcutsMutex,
		dirPathsToWatchers:  make(map[string]watcher.Watcher),
	}

	mainLoop(primary, primarySettingsWatcherConfig.Changes)
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

func mainLoop(primary *primarySettings, changes chan watcher.Changes) {
	for change := range changes {
		for _, filePath := range change.UpdatedFilePaths {
			log.Println("Main settings file has been updated:", filePath)

			switch path.Base(filePath) {
			case primary.app.Filename(""):
				err := primary.app.Reload(filePath)
				if err != nil {
					log.Println("Failed to load application settings -", err.Error())
					continue
				}

				updateGameCollectionWatchers(primary)
			case primary.launchers.Filename(""):
				err := primary.launchers.Reload(filePath)
				if err != nil {
					log.Println("Failed to load application settings -", err.Error())
					continue
				}
			}
		}
	}
}

func updateGameCollectionWatchers(primary *primarySettings) {
	watchDirs := primary.app.WatchPaths()

	OUTER:
	for dirPath, currentWatcher := range primary.dirPathsToWatchers {
		for _, newDirPath := range watchDirs {
			if dirPath == newDirPath {
				continue OUTER
			}
		}

		log.Println("No longer watching", dirPath)

		currentWatcher.Destroy()

		delete(primary.dirPathsToWatchers, dirPath)
	}

	for _, dirPath := range watchDirs {
		_, ok := primary.dirPathsToWatchers[dirPath]
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

		primary.dirPathsToWatchers[dirPath] = w
	}
}
