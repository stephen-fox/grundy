package main

import (
	"flag"
	"log"
	"os"
	"path"
	"sync"

	"github.com/stephen-fox/grundy/internal/gcw"
	"github.com/stephen-fox/grundy/internal/settings"
	"github.com/stephen-fox/watcher"
)

const (
	appSettingsDirPathArg = "settings"
	helpArg               = "h"
)

type primarySettings struct {
	app                settings.AppSettings
	launchers          settings.LaunchersSettings
	steamShortuctsLock *sync.Mutex
	dirPathsToWatchers map[string]watcher.Watcher
}

var (
	appSettingsDirPath = flag.String(appSettingsDirPathArg, settings.DirPath(), "The directory to store application settings")
	help               = flag.Bool(helpArg, false, "Show this help information")
)

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

	primary := &primarySettings{
		app:                appSettings,
		launchers:          launchersSettings,
		steamShortuctsLock: &sync.Mutex{},
		dirPathsToWatchers: make(map[string]watcher.Watcher),
	}

	mainLoop(primary, mainWatcherConfig.Changes)
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
			AppSettingsDirPath: *appSettingsDirPath,
			DirPath:            dirPath,
			Launchers:          primary.launchers,
			SteamShortcutsLock: primary.steamShortuctsLock,
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
