package main

import (
	"errors"
	"flag"
	"log"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/stephen-fox/grundy/internal/settings"
	"github.com/stephen-fox/steamutil/locations"
	"github.com/stephen-fox/steamutil/shortcuts"
	"github.com/stephen-fox/watcher"
)

const (
	appSettingsDirPathArg = "settings"
	helpArg               = "h"
)

type steamShortcutsConfig struct {
	shortcutsFilePath string
	game              settings.GameSettings
	launcher          settings.Launcher
	fileAccess        *sync.Mutex
}

var (
	appSettingsDirPath = flag.String(appSettingsDirPathArg, settings.DirPath(), "The directory to store application settings")
	help               = flag.Bool(helpArg, false, "Show this help information")
)

// TODO: Need to cleanup function inputs and Steam file access.
func main() {
	flag.Parse()

	if *help {
		flag.PrintDefaults()
		os.Exit(0)
	}

	launchersSettings := settings.NewLaunchersSettings()
	launchersSettings.AddOrUpdate(settings.NewLauncher())
	appSettings := settings.NewAppSettings()
	gameSettings := settings.NewGameSettings()

	saveableToShouldCreateInSettingsDir := map[settings.SaveableSettings]bool{
		launchersSettings: true,
		appSettings:       true,
		gameSettings:      false,
	}

	for s, createInMainDir := range saveableToShouldCreateInSettingsDir {
		err := settings.Create(*appSettingsDirPath + "/examples", settings.ExampleSuffix, s)
		if err != nil {
			log.Fatal("Failed to create default application settings files - " + err.Error())
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

	mainWatcherConfig := watcher.Config{
		ScanFunc:    watcher.ScanFilesInDirectory,
		RootDirPath: *appSettingsDirPath,
		FileSuffix:  settings.FileExtension,
		Changes:     make(chan watcher.Changes),
	}

	mainSettingsWatcher, err := watcher.NewWatcher(mainWatcherConfig)
	if err != nil {
		log.Fatal("Failed to watch application settings directory for changes")
	}
	defer mainSettingsWatcher.Stop()

	mainSettingsWatcher.Start()

	mainLoop(appSettings, launchersSettings, mainWatcherConfig.Changes)
}

func mainLoop(app settings.AppSettings, launchers settings.LaunchersSettings, changes chan watcher.Changes) {
	dirPathsToWatchers := make(map[string]watcher.Watcher)
	steamShortcutsLock := &sync.Mutex{}

	for change := range changes {
		for _, filePath := range change.UpdatedFilePaths {
			log.Println("Main settings file has been updated:", filePath)

			switch path.Base(filePath) {
			case app.Filename(""):
				err := app.Reload(filePath)
				if err != nil {
					log.Println("Failed to load application settings -", err.Error())
					continue
				}

				updateSubDirWatchers(dirPathsToWatchers, app, launchers, steamShortcutsLock)
			case launchers.Filename(""):
				err := launchers.Reload(filePath)
				if err != nil {
					log.Println("Failed to load application settings -", err.Error())
					continue
				}
			}
		}
	}
}

func updateSubDirWatchers(dirPathsToWatchers map[string]watcher.Watcher, app settings.AppSettings, launchers settings.LaunchersSettings, steamShortcutsLock *sync.Mutex) {
	watchDirs := app.WatchPaths()

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

		subDirWatcher, err := createSubDirWatcher(dirPath, launchers, steamShortcutsLock)
		if err != nil {
			log.Println(err.Error())
			continue
		}

		log.Println("Now watching subdirectories in", dirPath)

		dirPathsToWatchers[dirPath] = subDirWatcher
	}
}

func createSubDirWatcher(dirPath string, launchers settings.LaunchersSettings, steamShortcutsLock *sync.Mutex) (watcher.Watcher, error) {
	watcherConfig := watcher.Config{
		ScanFunc:    watcher.ScanFilesInSubdirectories,
		RootDirPath: dirPath,
		FileSuffix:  settings.FileExtension,
		Changes:     make(chan watcher.Changes),
	}

	subDirWatcher, err := watcher.NewWatcher(watcherConfig)
	if err != nil {
		return nil, errors.New("Failed to create watcher for " + dirPath + " - " + err.Error())
	}

	subDirWatcher.Start()

	go subDirWatcherLoop(watcherConfig, launchers, steamShortcutsLock)

	return subDirWatcher, nil
}

func subDirWatcherLoop(config watcher.Config, launchers settings.LaunchersSettings, steamShortcutsLock *sync.Mutex) {
	for change := range config.Changes {
		if change.IsErr() {
			log.Println("An error occurred when getting changes for", config.RootDirPath, "-", change.Err)
			continue
		}

		verifier, err := locations.NewDataVerifier()
		if err != nil {
			log.Println("Failed to create Steam data verifier -", err.Error())
			continue
		}

		idsToDirs, err := verifier.UserIdsToDataDirPaths()
		if err != nil {
			log.Println("Failed to get Steam user ID directories -", err.Error())
			continue
		}

		// TODO: Use KnownGameSettings to figure out if a game was deleted and then remove the shortcut.

		for _, updated := range change.UpdatedFilePaths {
			if strings.HasPrefix(updated, *appSettingsDirPath) {
				continue
			}

			log.Println("Game settings", updated, "was updated")

			game, err := settings.LoadGameSettings(updated)
			if err != nil {
				log.Println("Failed to load game settings for", updated, "-", err.Error())
				continue
			}

			l, ok := launchers.Has(game.Launcher())
			if !ok {
				log.Println("The specified launcher does not exist in the launchers settings - " + game.Launcher())
				continue
			}

			for steamUserId := range idsToDirs {
				shortcutsPath := locations.ShortcutsFilePath(verifier.RootDirPath(), steamUserId)

				log.Println("Creating Steam shortcut for '" + game.Name() + "'...")

				config := steamShortcutsConfig{
					shortcutsFilePath: shortcutsPath,
					game:              game,
					launcher:          l,
					fileAccess:        steamShortcutsLock,
				}

				wasUpdated, err := createOrUpdateSteamShortcut(config)
				if err != nil {
					log.Println("Failed to create or update Steam shortcut for", game.Name(), "-", err.Error())
					continue
				}

				if wasUpdated {
					log.Println("Updated shortcut for", game.Name())
				} else {
					log.Println("Created shortcut for", game.Name())
				}
			}
		}
	}
}

func createOrUpdateSteamShortcut(config steamShortcutsConfig) (bool, error) {
	config.fileAccess.Lock()
	defer config.fileAccess.Unlock()

	var flags int
	var fileAlreadyExists bool

	_, statErr := os.Stat(config.shortcutsFilePath)
	if statErr == nil {
		flags = os.O_RDWR
		fileAlreadyExists = true
	} else {
		flags = os.O_CREATE|os.O_RDWR
	}

	f, err := os.OpenFile(config.shortcutsFilePath, flags, 0600)
	if err != nil {
		return false, errors.New("Failed to open Steam shortcuts file - " + err.Error())
	}
	defer f.Close()

	var scs []shortcuts.Shortcut

	if fileAlreadyExists {
		scs, err = shortcuts.Shortcuts(f)
		if err != nil {
			return false, err
		}

		_, err = f.Seek(0, 0)
		if err != nil {
			return false, err
		}
	}

	var options string

	if config.game.ShouldOverrideLauncherArgs() {
		options = config.game.LauncherOverrideArgs()
	} else {
		options = config.launcher.DefaultArgs() + " " + config.game.AdditionalLauncherArgs()
	}

	options = options + " " + config.game.ExePath(true)

	var updated bool

	for i := range scs {
		if scs[i].AppName == config.game.Name() {
			scs[i].StartDir = config.launcher.ExeDirPath()
			scs[i].ExePath = config.launcher.ExePath()
			scs[i].LaunchOptions = options
			scs[i].IconPath = config.game.IconPath()
			scs[i].Tags = config.game.Categories()

			updated = true
			break
		}
	}

	if !updated {
		s := shortcuts.Shortcut{
			Id:            len(scs),
			AppName:       config.game.Name(),
			ExePath:       config.launcher.ExePath(),
			StartDir:      config.launcher.ExeDirPath(),
			IconPath:      config.game.IconPath(),
			LaunchOptions: options,
			Tags:          config.game.Categories(),
		}

		scs = append(scs, s)
	}

	err = shortcuts.WriteVdfV1(scs, f)
	if err != nil {
		return false, err
	}

	return updated, nil
}
