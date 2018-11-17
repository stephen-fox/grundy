package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"github.com/stephen-fox/grundy/internal/daemonw"
	"github.com/stephen-fox/grundy/internal/installer"
	"github.com/stephen-fox/grundy/internal/lock"
	"github.com/stephen-fox/grundy/internal/settings"
	"github.com/stephen-fox/grundy/internal/shortman"
	"github.com/stephen-fox/grundy/internal/steamw"
	"github.com/stephen-fox/watcher"
)

const (
	name        = "grundy"
	description = "Grundy crushes your games into Steam shortcuts " +
		"so you do not have to! Please refer to the usage documentation " +
		"at https://github.com/stephen-fox/grundy for more information."

	installArg            = "install"
	uninstallArg          = "uninstall"
	daemonCommandArg      = "daemon"
	appSettingsDirPathArg = "settings"
	helpArg               = "h"
)

var (
	daemonId string
	version  string
)

type application struct {
	primary *primarySettings
	stop    chan chan struct{}
}

func (o *application) Start() error {
	log.Println("Starting...")

	go mainLoop(o.primary, o.stop)

	return nil
}

func (o *application) Stop() error {
	log.Println("Stopping...")

	c := make(chan struct{})
	o.stop <- c
	<-c

	log.Println("Finished stopping resources")

	return nil
}

type primarySettings struct {
	dirPath       string
	watcher       watcher.Watcher
	watcherConfig watcher.Config
	app           settings.AppSettings
	launchers     settings.LaunchersSettings
	knownGames    settings.KnownGamesSettings
}

func main() {
	doInstall := flag.Bool(installArg, false, "Installs the application")
	doUninstall := flag.Bool(uninstallArg, false, "Uninstalls the application")
	appSettingsDirPath := flag.String(appSettingsDirPathArg, settings.DirPath(),
		"The directory to store application settings")
	daemonCommand := flag.String(daemonCommandArg, "",
		"Manage the application's daemon with the following commands:\n" +
		daemonw.CommandsString())
	help := flag.Bool(helpArg, false, "Show this help information")

	flag.Parse()

	if *help {
		fmt.Println(name, version)
		flag.PrintDefaults()
		os.Exit(0)
	}

	daemonConfig, err := daemonw.GetConfig(daemonId, description)
	if err != nil {
		log.Fatal("Failed to create daemon config - " + err.Error())
	}

	if *doInstall {
		err := installer.Install(daemonConfig)
		if err != nil {
			log.Fatal(err.Error())
		}

		os.Exit(0)
	}

	if *doUninstall {
		err := installer.Uninstall(daemonConfig)
		if err != nil {
			log.Fatal(err.Error())
		}

		os.Exit(0)
	}

	if len(strings.TrimSpace(*daemonCommand)) > 0 {
		log.Println("Executing daemon command '" + *daemonCommand + "'...")

		output, err := daemonw.ExecuteCommand(daemonw.Command(*daemonCommand), daemonConfig)
		if err != nil {
			log.Fatal(err.Error())
		}

		if len(output) > 0 {
			log.Println(output)
		}

		os.Exit(0)
	}

	instanceLock := lock.NewLock(settings.InternalFilesDir(*appSettingsDirPath))
	err = instanceLock.Acquire()
	if err != nil {
		log.Fatal(err.Error())
	}
	defer instanceLock.Release()

	go func() {
		for err := range instanceLock.Errs() {
			log.Println("Error maintaining instance lock - " + err.Error())
		}
	}()

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

	app := &application{
		primary: primary,
		stop:    make(chan chan struct{}),
	}

	err = daemonw.BlockAndRun(app, daemonConfig)
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

	knownGames, loaded := settings.LoadOrCreateKnownGamesSettings(internalDirPath)
	if loaded {
		err := cleanupKnownGameShortcuts(knownGames)
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
		dirPath:       settingsDirPath,
		watcherConfig: primarySettingsWatcherConfig,
		watcher:       primarySettingsWatcher,
		app:           app,
		launchers:     launchers,
		knownGames:    knownGames,
	}, nil
}

func cleanupKnownGameShortcuts(knownGames settings.KnownGamesSettings) error {
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
		GameNames: targets,
		Info:      info,
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

	gameCollectionChanges := make(chan watcher.Changes)
	dirPathsToWatchers  := make(map[string]watcher.Watcher)

	shortcutManagerConfig := shortman.Config{
		KnownGames:       primary.knownGames,
		Launchers:        primary.launchers,
		IgnorePathPrefix: primary.dirPath,
	}

	shortcutManager := shortman.NewShortcutManager(shortcutManagerConfig)

	updateWatchersTimer := time.NewTimer(1 * time.Second)
	stopTimerSafely(updateWatchersTimer)

	refreshShortcutsTimer := time.NewTimer(1 * time.Second)
	stopTimerSafely(refreshShortcutsTimer)

	for {
		select {
		case change := <-primary.watcherConfig.Changes:
			processPrimarySettingsChange(change.UpdatedFilePaths, primary, refreshShortcutsTimer, updateWatchersTimer)
		case <-updateWatchersTimer.C:
			log.Println("Updating game collection watchers...")

			updateGameCollectionWatchers(primary, dirPathsToWatchers, gameCollectionChanges)
		case <-refreshShortcutsTimer.C:
			log.Println("Refreshing shortcuts for known games...")

			result, err := shortcutManager.RefreshAllShortcuts()
			if err != nil {
				log.Println("An error occurred when refreshing shortcuts for all games - " + err.Error())
				continue
			}

			logShortcutManagerResult(result)
		case change := <-gameCollectionChanges:
			if change.IsErr() {
				log.Println("Failed to get changes for game collection - " + change.Err.Error())
				continue
			}

			result, err := shortcutManager.UpdateShortcuts(change.UpdatedFilePaths, change.DeletedFilePaths)
			if err != nil {
				log.Println("Failed to create or update shortcuts - " + err.Error())
				continue
			}

			logShortcutManagerResult(result)

			if result.Created.IsErr() || result.Deleted.IsErr() {
				// TODO: Do something?
			}
		case c := <-stop:
			for k, w := range dirPathsToWatchers {
				w.Destroy()

				delete(dirPathsToWatchers, k)
			}

			primary.watcher.Destroy()

			c <- struct{}{}

			return
		}
	}
}

func processPrimarySettingsChange(updatedPaths []string, primary *primarySettings, refresh *time.Timer, updateWatchers *time.Timer) {
	timerDelay := 5 * time.Second

	for _, filePath := range updatedPaths {
		log.Println("Main settings file has been updated:", filePath)

		switch path.Base(filePath) {
		case primary.app.Filename(""):
			err := primary.app.Reload(filePath)
			if err != nil {
				log.Println("Failed to load application settings -", err.Error())
				continue
			}

			stopTimerSafely(updateWatchers)
			updateWatchers.Reset(timerDelay)
		case primary.launchers.Filename(""):
			err := primary.launchers.Reload(filePath)
			if err != nil {
				log.Println("Failed to load launchers settings -", err.Error())
				continue
			}

			stopTimerSafely(refresh)
			refresh.Reset(timerDelay)
		default:
			continue
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

func updateGameCollectionWatchers(primary *primarySettings, dirPathsToWatchers map[string]watcher.Watcher, changes chan watcher.Changes) {
	watchDirs := primary.app.WatchPaths()

OUTER:
	for dirPath, currentWatcher := range dirPathsToWatchers {
		for _, newDirPath := range watchDirs {
			if dirPath == newDirPath {
				continue OUTER
			}
		}

		log.Println("No longer watching", dirPath)

		currentWatcher.Stop()

		delete(dirPathsToWatchers, dirPath)
	}

	for _, dirPath := range watchDirs {
		_, ok := dirPathsToWatchers[dirPath]
		if ok {
			continue
		}

		collectionWatcherConfig := watcher.Config{
			ScanFunc:    watcher.ScanFilesInSubdirectories,
			RootDirPath: dirPath,
			FileSuffix:  settings.FileExtension,
			Changes:     changes,
		}

		w, err := watcher.NewWatcher(collectionWatcherConfig)
		if err != nil {
			log.Println("Failed to create game collection watcher for " +
				dirPath + " - " + err.Error())
			continue
		}

		log.Println("Now watching subdirectories in", dirPath)

		w.Start()

		dirPathsToWatchers[dirPath] = w
	}
}

func logShortcutManagerResult(result shortman.ManageResult) {
	for _, s := range result.Created.CreatedInfo() {
		log.Println(s)
	}

	for _, s := range result.Created.NotAddedInfo() {
		log.Println(s)
	}

	for _, s := range result.Created.UpdatedInfo() {
		log.Println(s)
	}

	for _, s := range result.Created.FailuresInfo() {
		log.Println(s)
	}

	for _, s := range result.Deleted.DeletedInfo() {
		log.Println(s)
	}

	for _, s := range result.Deleted.NotDeletedInfo() {
		log.Println(s)
	}

	for _, s := range result.Deleted.NotDeletedInfo() {
		log.Println(s)
	}
}
