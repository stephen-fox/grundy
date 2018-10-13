package main

import (
	"flag"
	"log"
	"os"
	"path"

	"github.com/stephen-fox/grundy/internal/settings"
	"github.com/stephen-fox/watcher"
)

const (
	appSettingsDirPathArg = "settings"
	helpArg               = "h"
)

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
			log.Fatal("Failed to create default application setting files - " + err.Error())
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

	loopAndUpdateSettings(appSettings, launchersSettings, mainWatcherConfig.Changes)
}

func loopAndUpdateSettings(app settings.AppSettings, launchers settings.LaunchersSettings, changes chan watcher.Changes) {
	m := make(map[string]watcher.Watcher)

	for change := range changes {
		for _, filePath := range change.UpdatedFilePaths {
			log.Println("Main setting file has been updated:", filePath)

			switch path.Base(filePath) {
			case app.Filename(""):
				err := app.Reload(filePath)
				if err != nil {
					log.Println("Failed to load application settings -", err.Error())
				}
			case launchers.Filename(""):
				err := launchers.Reload(filePath)
				if err != nil {
					log.Println("Failed to load application settings -", err.Error())
				}
			default:

			}
		}
	}
}
