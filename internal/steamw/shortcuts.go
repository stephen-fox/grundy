package steamw

import (
	"errors"
	"os"

	"github.com/stephen-fox/grundy/internal/settings"
	"github.com/stephen-fox/steamutil/locations"
	"github.com/stephen-fox/steamutil/shortcuts"
)

const (
	defaultShortcutsFileMode = 0644
)

type NewShortcutConfig struct {
	Game     settings.GameSettings
	Launcher settings.Launcher
	Info     DataInfo
}

type DeleteShortcutConfig struct {
	GameNames []string
	Info      DataInfo
}

type NewShortcutResult struct {
	CreatedForIds []string
	UpdatedForIds []string
	IdsToFailures map[string]error
}

type DeletedShortcutsForSteamIdsResult struct {
	IdsToDeletedGames map[string][]string
	IdsToNotDeletedGames map[string][]string
	IdsToFailures map[string]error
}

type DeletedShortcutResult struct {
	Deleted    []string
	NotDeleted []string
}

func CreateOrUpdateShortcutPerId(config NewShortcutConfig) NewShortcutResult {
	result := NewShortcutResult{
		IdsToFailures: make(map[string]error),
	}

	for steamUserId := range config.Info.IdsToDirPaths {
		shortcutsPath := locations.ShortcutsFilePath(config.Info.DataLocations.RootDirPath(), steamUserId)

		fileUpdateResult, err := CreateOrUpdateShortcut(config, shortcutsPath)
		if err != nil {
			result.IdsToFailures[steamUserId] = err
			continue
		}

		switch fileUpdateResult {
		case shortcuts.UpdatedEntry:
			result.UpdatedForIds = append(result.UpdatedForIds, steamUserId)
		default:
			result.CreatedForIds = append(result.CreatedForIds, steamUserId)
		}
	}

	return result
}

func CreateOrUpdateShortcut(config NewShortcutConfig, shortcutsFilePath string) (shortcuts.UpdateResult, error) {
	var options string

	if config.Game.ShouldOverrideLauncherArgs() {
		options = config.Game.LauncherOverrideArgs()
	} else {
		options = config.Launcher.DefaultArgs() + " " + config.Game.AdditionalLauncherArgs()
	}

	exePath, exists := config.Game.ExePath()
	if !exists {
		return shortcuts.Unchanged, errors.New("The game executable does not exist")
	}

	options = options + " " + exePath

	onMatch := func(name string, matched *shortcuts.Shortcut) {
		matched.StartDir = config.Launcher.ExeDirPath()
		matched.ExePath = config.Launcher.ExePath()
		matched.LaunchOptions = options
		matched.IconPath = config.Game.IconPath()
		matched.Tags = config.Game.Categories()
	}

	noMatch := func(name string) shortcuts.Shortcut {
		return shortcuts.Shortcut{
			AppName:       config.Game.Name(),
			ExePath:       config.Launcher.ExePath(),
			StartDir:      config.Launcher.ExeDirPath(),
			IconPath:      config.Game.IconPath(),
			LaunchOptions: options,
			Tags:          config.Game.Categories(),
		}
	}

	createOrUpdateConfig := shortcuts.CreateOrUpdateConfig{
		MatchName: config.Game.Name(),
		Path:      shortcutsFilePath,
		Mode:      defaultShortcutsFileMode,
		OnMatch:   onMatch,
		NoMatch:   noMatch,
	}

	result, err := shortcuts.CreateOrUpdateVdfV1File(createOrUpdateConfig)
	if err != nil {
		return result, err
	}

	return result, nil
}

func DeleteShortcutPerId(config DeleteShortcutConfig) DeletedShortcutsForSteamIdsResult {
	result := DeletedShortcutsForSteamIdsResult{
		IdsToDeletedGames: make(map[string][]string),
		IdsToNotDeletedGames: make(map[string][]string),
		IdsToFailures: make(map[string]error),
	}

	for steamUserId := range config.Info.IdsToDirPaths {
		shortcutsPath := locations.ShortcutsFilePath(config.Info.DataLocations.RootDirPath(), steamUserId)

		delResult, err := DeleteShortcuts(config, shortcutsPath)
		if err != nil {
			result.IdsToFailures[steamUserId] = err
			continue
		}

		if len(delResult.Deleted) > 0 {
			result.IdsToDeletedGames[steamUserId] = delResult.Deleted
		}

		if len(delResult.NotDeleted) > 0 {
			result.IdsToNotDeletedGames[steamUserId] =  delResult.NotDeleted
		}
	}

	return result
}

func DeleteShortcuts(config DeleteShortcutConfig, shortcutsFilePath string) (DeletedShortcutResult, error) {
	f, err := os.OpenFile(shortcutsFilePath, os.O_RDWR, defaultShortcutsFileMode)
	if err != nil {
		return DeletedShortcutResult{}, err
	}
	defer f.Close()

	scs, err := shortcuts.ReadVdfV1(f)
	if err != nil {
		return DeletedShortcutResult{}, err
	}

	_, err = f.Seek(0, 0)
	if err != nil {
		return DeletedShortcutResult{}, err
	}

	var deleted []string

	notDeleted := make([]string, len(config.GameNames))

	copy(notDeleted, config.GameNames)

	for shortcutIndex, s := range scs {
		for delIndex := range notDeleted {
			if notDeleted[delIndex] == s.AppName {
				scs = append(scs[:shortcutIndex], scs[shortcutIndex+1:]...)

				deleted = append(deleted, notDeleted[delIndex])

				notDeleted = append(notDeleted[:delIndex], notDeleted[delIndex+1:]...)

				break
			}
		}
	}

	err = f.Truncate(0)
	if err != nil {
		return DeletedShortcutResult{}, err
	}

	err = shortcuts.WriteVdfV1(scs, f)
	if err != nil {
		return DeletedShortcutResult{}, err
	}

	return DeletedShortcutResult{
		Deleted:    deleted,
		NotDeleted: notDeleted,
	}, nil
}
