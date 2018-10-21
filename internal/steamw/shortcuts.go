package steamw

import (
	"errors"
	"os"
	"sync"

	"github.com/stephen-fox/grundy/internal/settings"
	"github.com/stephen-fox/steamutil/locations"
	"github.com/stephen-fox/steamutil/shortcuts"
)

const (
	defaultShortcutsFileMode = 0644
)

type NewShortcutConfig struct {
	Game       settings.GameSettings
	Launcher   settings.Launcher
	Info       DataInfo
	FileAccess *sync.Mutex
}

type DeleteShortcutConfig struct {
	GameNames  []string
	Info       DataInfo
	FileAccess *sync.Mutex
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

		wasUpdated, err := CreateOrUpdateShortcut(config, shortcutsPath)
		if err != nil {
			result.IdsToFailures[steamUserId] = err
			continue
		}

		if wasUpdated {
			result.UpdatedForIds = append(result.UpdatedForIds, steamUserId)
		} else {
			result.CreatedForIds = append(result.CreatedForIds, steamUserId)
		}
	}

	return result
}

func CreateOrUpdateShortcut(config NewShortcutConfig, shortcutsFilePath string) (bool, error) {
	config.FileAccess.Lock()
	defer config.FileAccess.Unlock()

	//var flags int
	var fileAlreadyExists bool

	// TODO: This can probably be replaced with 'flags = os.O_RDWR|os.O_CREATE'.
	//_, statErr := os.Stat(shortcutsFilePath)
	//if statErr == nil {
	//	flags = os.O_RDWR
	//	fileAlreadyExists = true
	//} else {
	//	flags = os.O_CREATE|os.O_RDWR
	//}

	f, err := os.OpenFile(shortcutsFilePath, os.O_RDWR|os.O_CREATE, defaultShortcutsFileMode)
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

	if config.Game.ShouldOverrideLauncherArgs() {
		options = config.Game.LauncherOverrideArgs()
	} else {
		options = config.Launcher.DefaultArgs() + " " + config.Game.AdditionalLauncherArgs()
	}

	options = options + " " + config.Game.ExePath(true)

	var updated bool

	for i := range scs {
		if scs[i].AppName == config.Game.Name() {
			scs[i].StartDir = config.Launcher.ExeDirPath()
			scs[i].ExePath = config.Launcher.ExePath()
			scs[i].LaunchOptions = options
			scs[i].IconPath = config.Game.IconPath()
			scs[i].Tags = config.Game.Categories()

			updated = true
			break
		}
	}

	if !updated {
		s := shortcuts.Shortcut{
			Id:            len(scs),
			AppName:       config.Game.Name(),
			ExePath:       config.Launcher.ExePath(),
			StartDir:      config.Launcher.ExeDirPath(),
			IconPath:      config.Game.IconPath(),
			LaunchOptions: options,
			Tags:          config.Game.Categories(),
		}

		scs = append(scs, s)
	}

	err = shortcuts.WriteVdfV1(scs, f)
	if err != nil {
		return false, err
	}

	return updated, nil
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
	config.FileAccess.Lock()
	defer config.FileAccess.Unlock()

	f, err := os.OpenFile(shortcutsFilePath, os.O_RDWR, defaultShortcutsFileMode)
	if err != nil {
		return DeletedShortcutResult{}, err
	}
	defer f.Close()

	scs, err := shortcuts.Shortcuts(f)
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
