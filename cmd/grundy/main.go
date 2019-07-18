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
	"github.com/stephen-fox/grundy/internal/results"
	"github.com/stephen-fox/grundy/internal/settings"
	"github.com/stephen-fox/grundy/internal/shortman"
	"github.com/stephen-fox/grundy/internal/steamw"
	"github.com/stephen-fox/ipcm"
	"github.com/stephen-fox/watcher"
)

type configReloadAction string

const (
	refreshKnownGames     configReloadAction = "refresh_known_games"
	updateGameCollections configReloadAction = "update_game_collections"
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
	settings *settingsState
	stop     chan chan struct{}
}

func (o *application) Start() error {
	logInfo("Starting...")

	go mainLoop(o.settings, o.stop)

	return nil
}

func (o *application) Stop() error {
	logInfo("Stopping...")

	c := make(chan struct{})
	o.stop <- c
	<-c

	logInfo("Finished stopping resources")

	return nil
}

type settingsState struct {
	configDirPath    string
	watcher          watcher.Watcher
	configDirChanges chan watcher.Change
	app              settings.AppSettings
	launchers        settings.LaunchersSettings
	knownGames       settings.KnownGamesSettings
}

func (o *settingsState) reload(updatedPaths []string) map[configReloadAction]configReloadAction {
	actions := make(map[configReloadAction]configReloadAction)

	for _, filePath := range updatedPaths {
		logInfo("Settings file has been updated:", filePath)

		switch path.Base(filePath) {
		case o.app.Filename(""):
			err := o.app.Reload(filePath)
			if err != nil {
				logError("Failed to load application settings -", err.Error())
				continue
			}

			actions[updateGameCollections] = updateGameCollections
		case o.launchers.Filename(""):
			err := o.launchers.Reload(filePath)
			if err != nil {
				logError("Failed to load launchers settings -", err.Error())
				continue
			}

			actions[updateGameCollections] = updateGameCollections
			actions[refreshKnownGames] = refreshKnownGames
		default:
			continue
		}
	}

	return actions
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
		logFatal("Failed to create daemon config - " + err.Error())
	}

	daemon, err := cyberdaemon.NewDaemon(daemonConfig)
	if err != nil {
		logFatal("Failed to create daemon - " + err.Error())
	}

	if *doInstall {
		err := installer.Install(daemon)
		if err != nil {
			logFatal(err.Error())
		}

		os.Exit(0)
	}

	if *doUninstall {
		err := installer.Uninstall(daemon)
		if err != nil {
			logFatal(err.Error())
		}

		os.Exit(0)
	}

	if len(strings.TrimSpace(*daemonCommand)) > 0 {
		logInfo("Executing daemon command '" + *daemonCommand + "'...")

		output, err := daemon.ExecuteCommand(cyberdaemon.Command(*daemonCommand))
		if err != nil {
			logFatal(err.Error())
		}

		if len(output) > 0 {
			logInfo(output)
		}

		os.Exit(0)
	}

	appMutex, err := ipcm.NewMutex(ipcm.MutexConfig{
		Resource: path.Join(settings.InternalFilesDir(*appSettingsDirPath), "lock"),
	})
	if err != nil {
		logFatal(err.Error())
	}

	err = appMutex.TimedTryLock(3 * time.Second)
	if err != nil {
		logFatal("another instance of the application is running - ", err.Error())
	}
	defer appMutex.Unlock()

	logFile, err := settings.LogFile(*appSettingsDirPath)
	if err != nil {
		logFatal(err.Error())
	}
	defer logFile.Close()

	log.SetOutput(io.MultiWriter(logFile, os.Stderr))

	currentSettings, err := setupSettings(*appSettingsDirPath)
	if err != nil {
		logFatal(err.Error())
	}

	app := &application{
		settings: currentSettings,
		stop:     make(chan chan struct{}),
	}

	err = daemon.BlockAndRun(app)
	if err != nil {
		logFatal(err.Error())
	}
}

func setupSettings(settingsDirPath string) (*settingsState, error) {
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
			return nil, errors.New("Failed to create example application settings file - " + err.Error())
		}

		if createInMainDir {
			_, statErr := os.Stat(path.Join(settingsDirPath, s.Filename("")))
			if statErr != nil {
				err := settings.Create(settingsDirPath, "", s)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	internalDirPath, err := settings.CreateInternalFilesDir(settingsDirPath)
	if err != nil {
		return nil, errors.New("Failed to create internal settings directory path - " + err.Error())
	}

	knownGames, loaded := settings.LoadOrCreateKnownGamesSettings(internalDirPath)
	if loaded {
		err := cleanupKnownGameShortcuts(knownGames)
		if err != nil {
			logError("Failed to cleanup known game shortcuts -", err.Error())
		}
	}

	configDirWatcherConfig := watcher.Config{
		ScanFunc:     watcher.ScanFilesInDirectory,
		RootDirPath:  settingsDirPath,
		FileSuffixes: []string{settings.FileExtension},
		Changes:      make(chan watcher.Change),
	}

	configDirWatcher, err := watcher.NewWatcher(configDirWatcherConfig)
	if err != nil {
		return nil, errors.New("Failed to watch application settings directory for changes - " + err.Error())
	}

	return &settingsState{
		configDirPath:    settingsDirPath,
		configDirChanges: configDirWatcherConfig.Changes,
		watcher:          configDirWatcher,
		app:              app,
		launchers:        launchers,
		knownGames:       knownGames,
	}, nil
}

// cleanupKnownGameShortcuts removes any shortcuts for games that we do not
// know about anymore.
//
// This function is not in 'shortman' because it is more efficient to run it
// early on (before we create a 'ShortcutManager'.
func cleanupKnownGameShortcuts(knownGames settings.KnownGamesSettings) error {
	gameDirPathsToGameNames := knownGames.DisownNonExistingGames()
	if len(gameDirPathsToGameNames) == 0 {
		return nil
	}

	info, err := steamw.NewSteamDataInfo()
	if err != nil {
		return err
	}

	for _, gameName := range gameDirPathsToGameNames {
		config := steamw.DeleteShortcutConfig{
			GameName:            gameName,
			Info:                info,
			SkipGridImageDelete: true,
		}

		deleteResults := steamw.DeleteShortcut(config)

		for i := range deleteResults {
			logResult(deleteResults[i])
		}
	}

	return nil
}

func mainLoop(currentSettings *settingsState, stop chan chan struct{}) {
	currentSettings.watcher.Start()

	gameCollectionChanges := make(chan watcher.Change)
	dirPathsToWatchers  := make(map[string]watcher.Watcher)

	shortcutManager := shortman.NewShortcutManager(shortman.Config{
		App:              currentSettings.app,
		KnownGames:       currentSettings.knownGames,
		Launchers:        currentSettings.launchers,
		IgnorePathPrefix: currentSettings.configDirPath,
	})

	updateCollectionsTimer := newStoppedTimer()

	refreshKnownGamesTimer := newStoppedTimer()

	timerDuration := 5 * time.Second

	for {
		select {
		case configDirChange := <-currentSettings.configDirChanges:
			if configDirChange.IsErr() {
				continue
			}

			for _, action := range currentSettings.reload(configDirChange.UpdatedFilePaths()) {
				switch action {
				case updateGameCollections:
					stopTimerSafely(updateCollectionsTimer)
					updateCollectionsTimer.Reset(timerDuration)
				case refreshKnownGames:
					stopTimerSafely(refreshKnownGamesTimer)
					refreshKnownGamesTimer.Reset(timerDuration)
				default:
					logError("Unknown action: ", action)
				}
			}
		case <-updateCollectionsTimer.C:
			logInfo("Updating game collections...")

			updateGameCollectionWatchers(currentSettings, dirPathsToWatchers, gameCollectionChanges)
		case <-refreshKnownGamesTimer.C:
			logInfo("Refreshing known games and their shortcuts...")

			steamDataInfo, err := steamw.NewSteamDataInfo()
			if err != nil {
				logError("Failed to get Steam info - " + err.Error())
				continue
			}

			for _, r := range shortcutManager.RefreshAll(steamDataInfo) {
				logResult(r)
			}
		case collectionChange := <-gameCollectionChanges:
			if collectionChange.IsErr() {
				logError("Failed to get changes for game collection - " + collectionChange.ErrDetails())

				// TODO: Delete all shortcuts if the collection no longer exists?
				continue
			}

			steamDataInfo, err := steamw.NewSteamDataInfo()
			if err != nil {
				logError("Failed to get Steam info - " + err.Error())
				continue
			}

			// TODO: The following lines determine way too much about game collections'
			//  fates based on game icons. The code should determine game collection
			//  updates and deletions based on game files, rather than icon files.
			//  This code is unintuitive, and might lead to bugs when the business
			//  logic changes.

			updatedFilePaths := collectionChange.UpdatedFilePaths()
			// If a grid image or icon is deleted, add the path to
			// the list of updated file paths.
			updatedFilePaths = append(updatedFilePaths,
				collectionChange.DeletedFilePathsWithSuffixes(settings.GameImageSuffixes)...)

			res := shortcutManager.Update(updatedFilePaths, false, steamDataInfo)

			// Do not delete game collections if a game image is deleted.
			res = append(res, shortcutManager.Delete(
				collectionChange.DeletedFilePathsWithoutSuffixes(settings.GameImageSuffixes),
				false, steamDataInfo)...)

			for _, r := range res {
				logResult(r)
			}
		case rejoin := <-stop:
			for k, w := range dirPathsToWatchers {
				w.Destroy()

				delete(dirPathsToWatchers, k)
			}

			currentSettings.watcher.Destroy()

			rejoin <- struct{}{}

			return
		}
	}
}

func newStoppedTimer() *time.Timer {
	t := time.NewTimer(1*time.Second)
	stopTimerSafely(t)
	return t
}

func stopTimerSafely(t *time.Timer) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
}

func updateGameCollectionWatchers(currentSettings *settingsState, dirPathsToWatchers map[string]watcher.Watcher, changes chan watcher.Change) {
	gameCollectionsToLauncherNames := currentSettings.app.GameCollectionsPathsToLauncherNames()

	// Stop watchers for game collection directories we are no longer watching.
	for dirPath, currentWatcher := range dirPathsToWatchers {
		_, isWatchingDir := gameCollectionsToLauncherNames[dirPath]
		if isWatchingDir {
			continue
		}

		logInfo("No longer watching", dirPath)

		currentWatcher.Stop()

		delete(dirPathsToWatchers, dirPath)
	}

	// Create and start new game collection watchers.
	for collectionDirPath, launcherName := range gameCollectionsToLauncherNames {
		launcher, hasLauncher := currentSettings.launchers.Has(launcherName)
		if !hasLauncher {
			logError("The collection '" + collectionDirPath + "' will not be added - Launcher '" +
				launcher.Name() + "' does not exist in the launchers configuration file")
			continue
		}

		err := launcher.IsValid()
		if err != nil {
			logError("The collection '" + collectionDirPath +
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
			FileSuffixes: append(launcher.GameFileSuffixes(), settings.GameImageSuffixes...),
			Changes:      changes,
		}

		w, err = watcher.NewWatcher(collectionWatcherConfig)
		if err != nil {
			logError("Failed to create game collection watcher for " +
				collectionDirPath + " - " + err.Error())
			continue
		}

		logInfo("Now watching '" + collectionDirPath +"' as a game collection")

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

func logResult(result results.Result) {
	switch result.Outcome() {
	case results.SucceededWithWarning:
		logWarn(result.PrintableResult())
	case results.Failed:
		logError(result.PrintableResult())
	case results.Succeeded:
		fallthrough
	case results.Skipped:
		fallthrough
	default:
		logInfo(result.PrintableResult())
	}
}

func logError(v ...interface{}) {
	v = append([]interface{}{"[ERROR]"}, v...)
	log.Println(v...)
}

func logInfo(v ...interface{}) {
	v = append([]interface{}{"[INFO]"}, v...)
	log.Println(v...)
}

func logWarn(v ...interface{}) {
	v = append([]interface{}{"[WARN]"}, v...)
	log.Println(v...)
}

func logFatal(v ...interface{}) {
	v = append([]interface{}{"[FATAL] "}, v...)
	log.Fatal(v...)
}
