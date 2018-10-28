package shortman

import (
	"os"
	"strings"

	"github.com/stephen-fox/grundy/internal/settings"
	"github.com/stephen-fox/grundy/internal/steamw"
)

type ShortcutManager interface {
	UpdateShortcuts(updatedConfigs []string, deletedConfigs []string) (ManageResult, error)
	RefreshAllShortcuts() (ManageResult, error)
}

type defaultShortcutManager struct {
	config Config
}

func (o *defaultShortcutManager) RefreshAllShortcuts() (ManageResult, error) {
	var deletedFilePaths []string
	var existingFilePaths []string

	for filePath := range o.config.KnownGames.ConfigFilePathsToGameNames() {
		_, statErr := os.Stat(filePath)
		if statErr != nil {
			deletedFilePaths = append(deletedFilePaths, filePath)
		} else {
			existingFilePaths = append(existingFilePaths, filePath)
		}
	}

	return o.UpdateShortcuts(existingFilePaths, deletedFilePaths)
}


func (o *defaultShortcutManager) UpdateShortcuts(updatedConfigs []string, deletedConfigs []string) (ManageResult, error) {
	info, err := steamw.NewSteamDataInfo()
	if err != nil {
		return ManageResult{}, err
	}

	return ManageResult{
		Deleted: o.deleteShortcuts(deletedConfigs, info),
		Created: o.createOrUpdateShortcuts(updatedConfigs, info),
	}, nil
}

func (o *defaultShortcutManager) deleteShortcuts(filePaths []string, info steamw.DataInfo) Deleted {
	var d Deleted

	for _, deleted := range filePaths {
		if strings.HasPrefix(deleted, o.config.IgnorePathPrefix) {
			continue
		}

		gameName, ok := o.config.KnownGames.Remove(deleted)
		if ok {
			config := steamw.DeleteShortcutConfig{
				GameNames: []string{gameName},
				Info:      info,
			}

			d.results = append(d.results, steamw.DeleteShortcutPerId(config))
		}
	}

	return d
}

func (o *defaultShortcutManager) createOrUpdateShortcuts(filePaths []string, info steamw.DataInfo) CreatedOrUpdated {
	c := CreatedOrUpdated{
		gameNamesToResults:    make(map[string]steamw.NewShortcutResult),
		configPathsToLoadErrs: make(map[string]error),
		notAddedToReasons:     make(map[string]string),
	}

	for _, updated := range filePaths {
		if strings.HasPrefix(updated, o.config.IgnorePathPrefix) {
			continue
		}

		game, err := settings.LoadGameSettings(updated)
		if err != nil {
			c.configPathsToLoadErrs[updated] = err
			continue
		}

		l, ok := o.config.Launchers.Has(game.Launcher())
		if !ok {
			c.notAddedToReasons[updated] = "The specified launcher does not " +
				"exist in the launchers settings - '" + game.Launcher() + "'"
			continue
		}

		ok = o.config.KnownGames.AddUniqueGameOnly(game, updated)
		if !ok {
			c.notAddedToReasons[updated] = "The game '" + game.Name() + "' already exists"
			continue
		}

		config := steamw.NewShortcutConfig{
			Game:     game,
			Launcher: l,
			Info:     info,
		}

		c.gameNamesToResults[game.Name()] = steamw.CreateOrUpdateShortcutPerId(config)
	}

	return c
}

type Config struct {
	KnownGames       settings.KnownGamesSettings
	Launchers        settings.LaunchersSettings
	IgnorePathPrefix string
}

func NewShortcutManager(config Config) ShortcutManager {
	return &defaultShortcutManager{
		config: config,
	}
}
