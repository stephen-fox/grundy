package gcw

import (
	"errors"
	"log"
	"strings"
	"sync"

	"github.com/stephen-fox/grundy/internal/settings"
	"github.com/stephen-fox/grundy/internal/steamw"
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

		info, err := steamw.NewSteamDataInfo()
		if err != nil {
			log.Println("Failed to get Steam information -", err.Error())
			continue
		}

		o.deleteShortcuts(change.DeletedFilePaths, info)

		o.createOrUpdateShortcuts(change.UpdatedFilePaths, info)
	}
}

func (o *gameCollectionWatcher) deleteShortcuts(filePaths []string, info steamw.DataInfo) {
	for _, deleted := range filePaths {
		if strings.HasPrefix(deleted, o.config.AppSettingsDirPath) {
			continue
		}

		log.Println("Game settings file", deleted, "was deleted")

		gameName, ok := o.config.KnownGames.Remove(deleted)
		if ok {
			log.Println("Deleting shortcut for", gameName + "...")

			config := steamw.DeleteShortcutConfig{
				GameNames:  []string{gameName},
				Info:       info,
				FileAccess: o.config.SteamShortcutsMutex,
			}

			result := steamw.DeleteShortcutPerId(config)

			for id, deleted := range result.IdsToDeletedGames {
				log.Println("Deleted shortcut for", deleted, "for Steam ID", id)
			}

			for id, notDeleted := range result.IdsToNotDeletedGames {
				log.Println("Shortcut for", notDeleted, "does not exist for Steam ID", id)
			}

			for id, err := range result.IdsToFailures {
				log.Println("Failed to delete shortcut for Steam user ID", id, "-", err.Error())
			}
		}
	}
}

func (o *gameCollectionWatcher) createOrUpdateShortcuts(filePaths []string, info steamw.DataInfo) {
	for _, updated := range filePaths {
		if strings.HasPrefix(updated, o.config.AppSettingsDirPath) {
			continue
		}

		log.Println("Game settings file '" + updated + "' was updated")

		game, err := settings.LoadGameSettings(updated)
		if err != nil {
			log.Println("Failed to load game settings for",
				updated, "-", err.Error())
			continue
		}

		l, ok := o.config.Launchers.Has(game.Launcher())
		if !ok {
			log.Println("The specified launcher does not " +
				"exist in the launchers settings - '" + game.Launcher() + "'")
			continue
		}

		ok = o.config.KnownGames.AddUniqueGameOnly(game, updated)
		if !ok {
			log.Println("The game '" + game.Name() +
				"' already exists, and will not be added to Steam")
			continue
		}

		config := steamw.NewShortcutConfig{
			Game:       game,
			Launcher:   l,
			Info:       info,
			FileAccess: o.config.SteamShortcutsMutex,
		}

		log.Println("Creating Steam shortcut for '" + game.Name() + "'...")

		result := steamw.CreateOrUpdateShortcutPerId(config)

		for _, c := range result.CreatedForIds {
			log.Println("Created shortcut to", game.Name(), "for Steam user ID", c)
		}

		for _, u := range result.UpdatedForIds {
			log.Println("Updated shortcut to", game.Name(), "for Steam user ID", u)
		}

		for f, err := range result.IdsToFailures {
			log.Println("Failed to create shortcut to", game.Name(),
				"for Steam user ID", f, "-", err.Error())
		}
	}
}

type WatcherConfig struct {
	AppSettingsDirPath  string
	DirPath             string
	Launchers           settings.LaunchersSettings
	KnownGames          settings.KnownGamesSettings
	SteamShortcutsMutex *sync.Mutex
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
