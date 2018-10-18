package gcw

import (
	"errors"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/stephen-fox/grundy/internal/settings"
	"github.com/stephen-fox/steamutil/locations"
	"github.com/stephen-fox/steamutil/shortcuts"
	"github.com/stephen-fox/watcher"
)

type gameCollectionWatcher struct {
	watcher watcher.Watcher
	config  *WatcherConfig
	changes chan watcher.Changes
}

func (o *gameCollectionWatcher) Start() {
	o.watcher.Start()
	go o.gameCollectionLooper()
}

func (o *gameCollectionWatcher) Stop() {
	o.watcher.Stop()
}

func (o *gameCollectionWatcher) Destroy() {
	o.watcher.Destroy()
}

func (o *gameCollectionWatcher) gameCollectionLooper() {
	for change := range o.changes {
		if change.IsErr() {
			log.Println("An error occurred when getting changes for",
				o.config.DirPath, "-", change.Err)
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

		data := steamData{
			dataLocations: verifier,
			idsToDirPaths: idsToDirs,
		}

		// TODO: Use KnownGameSettings to figure out if a game was deleted and then remove the shortcut.

		o.createOrUpdateShortcuts(change.UpdatedFilePaths, data)
	}
}

func (o *gameCollectionWatcher) createOrUpdateShortcuts(filePaths []string, data steamData) []settings.GameSettings {
	var games []settings.GameSettings

	for _, updated := range filePaths {
		if strings.HasPrefix(updated, o.config.AppSettingsDirPath) {
			continue
		}

		log.Println("Game settings", updated, "was updated")

		game, err := settings.LoadGameSettings(updated)
		if err != nil {
			log.Println("Failed to load game settings for", updated, "-", err.Error())
			continue
		}

		l, ok := o.config.Launchers.Has(game.Launcher())
		if !ok {
			log.Println("The specified launcher does not exist in the launchers settings - " + game.Launcher())
			continue
		}

		o.createOrUpdateSteamShortcutPerId(game, l, data)

		games = append(games, game)
	}

	return games
}

func (o *gameCollectionWatcher) createOrUpdateSteamShortcutPerId(game settings.GameSettings, l settings.Launcher, data steamData) {
	for steamUserId := range data.idsToDirPaths {
		shortcutsPath := locations.ShortcutsFilePath(data.dataLocations.RootDirPath(), steamUserId)

		log.Println("Creating Steam shortcut for '" + game.Name() + "'...")

		config := steamShortcutConfig{
			shortcutsFilePath: shortcutsPath,
			game:              game,
			launcher:          l,
			fileAccess:        o.config.SteamShortcutsLock,
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

type WatcherConfig struct {
	AppSettingsDirPath string
	DirPath            string
	Launchers          settings.LaunchersSettings
	SteamShortcutsLock *sync.Mutex
}

type steamData struct {
	dataLocations locations.DataVerifier
	idsToDirPaths map[string]string
}

type steamShortcutConfig struct {
	shortcutsFilePath string
	game              settings.GameSettings
	launcher          settings.Launcher
	fileAccess        *sync.Mutex
}

func NewGameCollectionWatcher(config *WatcherConfig) (watcher.Watcher, error) {
	watcherConfig := watcher.Config{
		ScanFunc:    watcher.ScanFilesInSubdirectories,
		RootDirPath: config.DirPath,
		FileSuffix:  settings.FileExtension,
		Changes:     make(chan watcher.Changes),
	}

	w, err := watcher.NewWatcher(watcherConfig)
	if err != nil {
		return nil, errors.New("Failed to create watcher for " + config.DirPath + " - " + err.Error())
	}

	return &gameCollectionWatcher{
		watcher: w,
		config:  config,
		changes: watcherConfig.Changes,
	}, nil
}

func createOrUpdateSteamShortcut(config steamShortcutConfig) (bool, error) {
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
