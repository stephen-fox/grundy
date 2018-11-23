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

	"github.com/stephen-fox/grundy/internal/cyberdaemon"
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
	configChanges chan watcher.Change
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
		cyberdaemon.CommandsString())
	help := flag.Bool(helpArg, false, "Show this help information")

	flag.Parse()

	if *help {
		fmt.Println(name, version)
		flag.PrintDefaults()
		os.Exit(0)
	}

	daemonConfig, err := cyberdaemon.GetConfig(daemonId, description)
	if err != nil {
		log.Fatal("Failed to create daemon config - " + err.Error())
	}

	daemon, err := cyberdaemon.NewDaemon(daemonConfig)
	if err != nil {
		log.Fatal("Failed to create daemon - " + err.Error())
	}

	if *doInstall {
		err := installer.Install(daemon)
		if err != nil {
			log.Fatal(err.Error())
		}

		os.Exit(0)
	}

	if *doUninstall {
		err := installer.Uninstall(daemon)
		if err != nil {
			log.Fatal(err.Error())
		}

		os.Exit(0)
	}

	if len(strings.TrimSpace(*daemonCommand)) > 0 {
		log.Println("Executing daemon command '" + *daemonCommand + "'...")

		output, err := daemon.ExecuteCommand(cyberdaemon.Command(*daemonCommand))
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

	err = daemon.BlockAndRun(app)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func setupPrimarySettings(settingsDirPath string) (*primarySettings, error) {
	launchers := settings.NewLaunchersSettings()
	launchers.AddOrUpdate(settings.NewLauncher())
	app := settings.NewAppSettings()
	exampleGame := settings.NewGameSettings("").Example()

	saveableToShouldCreateInSettingsDir := map[settings.SaveableSettings]bool{
		launchers:   true,
		app:         true,
		exampleGame: false,
	}

	for s, createInMainDir := range saveableToShouldCreateInSettingsDir {
		err := settings.Create(settingsDirPath + "/examples", settings.ExampleSuffix, s.Example())
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
		ScanFunc:     watcher.ScanFilesInDirectory,
		RootDirPath:  settingsDirPath,
		FileSuffixes: []string{settings.FileExtension},
		Changes:      make(chan watcher.Change),
	}

	primarySettingsWatcher, err := watcher.NewWatcher(primarySettingsWatcherConfig)
	if err != nil {
		return &primarySettings{}, errors.New("Failed to watch application settings directory for changes - " + err.Error())
	}

	return &primarySettings{
		dirPath:       settingsDirPath,
		configChanges: primarySettingsWatcherConfig.Changes,
		watcher:       primarySettingsWatcher,
		app:           app,
		launchers:     launchers,
		knownGames:    knownGames,
	}, nil
}

// TODO: Maybe move this into 'shortman'?
func cleanupKnownGameShortcuts(knownGames settings.KnownGamesSettings) error {
	gameDirPathsToGameNames := knownGames.DisownNonExistingGames()
	if len(gameDirPathsToGameNames) == 0 {
		return nil
	}

	var targets []string

	for _, gameName := range gameDirPathsToGameNames {
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
		log.Println("Shortcut for", notDeleted, "does not exist for Steam ID", id)
	}

	for id, err := range result.IdsToFailures {
		log.Println("Failed to cleanup shortcut for Steam user ID", id, "-", err.Error())
	}

	return nil
}

func mainLoop(primary *primarySettings, stop chan chan struct{}) {
	primary.watcher.Start()

	gameCollectionChanges := make(chan watcher.Change)
	dirPathsToWatchers  := make(map[string]watcher.Watcher)

	shortcutManagerConfig := shortman.Config{
		App:              primary.app,
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
		case configChange := <-primary.configChanges:
			if configChange.IsErr() {
				continue
			}

			processPrimarySettingsChange(configChange.UpdatedFilePaths(), primary, refreshShortcutsTimer, updateWatchersTimer)
		case <-updateWatchersTimer.C:
			log.Println("Updating game collection watchers...")

			updateGameCollectionWatchers(primary, dirPathsToWatchers, gameCollectionChanges)
		case <-refreshShortcutsTimer.C:
			log.Println("Refreshing shortcuts for known games...")

			steamDataInfo, err := steamw.NewSteamDataInfo()
			if err != nil {
				log.Println("Failed to get Steam info - " + err.Error())
				continue
			}

			createdUpdated, deleted := shortcutManager.RefreshAll(steamDataInfo)
			if err != nil {
				log.Println("An error occurred when refreshing shortcuts for all games - " + err.Error())
				continue
			}

			logShortcutManagerCreatedOrUpdated(createdUpdated)
			logShortcutManagerDeleted(deleted)
		case collectionChange := <-gameCollectionChanges:
			if collectionChange.IsErr() {
				log.Println("Failed to get changes for game collection - " + collectionChange.ErrDetails())

				// TODO: Delete all shortcuts if the collection no longer exists?
				continue
			}

			steamDataInfo, err := steamw.NewSteamDataInfo()
			if err != nil {
				log.Println("Failed to get Steam info - " + err.Error())
				continue
			}

			createdUpdated := shortcutManager.Update(collectionChange.UpdatedFilePaths(), false, steamDataInfo)

			deleted := shortcutManager.Delete(collectionChange.DeletedFilePaths(), false, steamDataInfo)

			logShortcutManagerCreatedOrUpdated(createdUpdated)

			logShortcutManagerDeleted(deleted)
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

func processPrimarySettingsChange(updatedPaths []string, primary *primarySettings, refreshShortcuts *time.Timer, updateWatchers *time.Timer) {
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

			stopTimerSafely(updateWatchers)
			updateWatchers.Reset(timerDelay)

			stopTimerSafely(refreshShortcuts)
			refreshShortcuts.Reset(timerDelay)
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

func updateGameCollectionWatchers(primary *primarySettings, dirPathsToWatchers map[string]watcher.Watcher, changes chan watcher.Change) {
	gameCollectionsToLauncherNames := primary.app.GameCollectionsPathsToLauncherNames()

	// Stop watchers for game collection directories we are no longer watching.
OUTER:
	for dirPath, currentWatcher := range dirPathsToWatchers {
		for newDirPath := range gameCollectionsToLauncherNames {
			if dirPath == newDirPath {
				continue OUTER
			}
		}

		log.Println("No longer watching", dirPath)

		currentWatcher.Stop()

		delete(dirPathsToWatchers, dirPath)
	}

	// Create and start new game collection watchers.
	for collectionDirPath, launcherName := range gameCollectionsToLauncherNames {
		launcher, hasLauncher := primary.launchers.Has(launcherName)
		if !hasLauncher {
			log.Println("The collection '" + collectionDirPath + "' will not be added - Launcher '" +
				launcher.Name() + "' does not exist in the launchers configuration file")
			continue
		}

		err := launcher.IsValid()
		if err != nil {
			log.Println("The collection '" + collectionDirPath +
				"' will not be added - The launcher is invalid - " + err.Error())
			continue
		}

		w, hasWatcher := dirPathsToWatchers[collectionDirPath]
		if hasWatcher && areSlicesEqual(w.Config().FileSuffixes, launcher.GameFileSuffixes()) {
			continue
		}

		collectionWatcherConfig := watcher.Config{
			ScanFunc:     watcher.ScanFilesInSubdirectories,
			RootDirPath:  collectionDirPath,
			FileSuffixes: launcher.GameFileSuffixes(),
			Changes:      changes,
		}

		w, err = watcher.NewWatcher(collectionWatcherConfig)
		if err != nil {
			log.Println("Failed to create game collection watcher for " +
				collectionDirPath + " - " + err.Error())
			continue
		}

		log.Println("Now watching '" + collectionDirPath +"' as a game collection")

		w.Start()

		dirPathsToWatchers[collectionDirPath] = w
	}
}

func areSlicesEqual(a[]string , b []string) bool {
	if (a == nil) != (b == nil) {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func logShortcutManagerCreatedOrUpdated(result shortman.CreatedOrUpdated) {
	for _, s := range result.CreatedInfo() {
		log.Println(s)
	}

	for _, s := range result.NotAddedInfo() {
		log.Println(s)
	}

	for _, s := range result.UpdatedInfo() {
		log.Println(s)
	}

	for _, s := range result.FailuresInfo() {
		log.Println(s)
	}
}

func logShortcutManagerDeleted(result shortman.Deleted) {
	for _, s := range result.DeletedInfo() {
		log.Println(s)
	}

	for _, s := range result.NotDeletedInfo() {
		log.Println(s)
	}

	for _, s := range result.FailedToDeleteInfo() {
		log.Println(s)
	}
}
