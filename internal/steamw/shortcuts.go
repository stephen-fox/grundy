package steamw

import (
	"os"
	"path"
	"strings"

	"github.com/stephen-fox/grundy/internal/results"
	"github.com/stephen-fox/steamutil/grid"
	"github.com/stephen-fox/steamutil/locations"
	"github.com/stephen-fox/steamutil/shortcuts"
)

const (
	defaultShortcutsFileMode = 0644
)

type NewShortcutConfig struct {
	Name          string
	LaunchOptions string
	ExePath       string
	IconPath      string
	TilePath      string
	Tags          []string
	Info          DataInfo
	Warnings      []string
}

type DeleteShortcutConfig struct {
	SkipTileDelete  bool
	LauncherExePath string
	GameName        string
	Info            DataInfo
}

type deleteShortcutResult struct {
	wasDeleted bool
}

func CreateOrUpdateShortcut(config NewShortcutConfig) []results.Result {
	var r []results.Result

	for steamUserId := range config.Info.IdsToDirPaths {
		shortcutsPath := locations.ShortcutsFilePath(config.Info.DataLocations.RootDirPath(), steamUserId)

		fileUpdateResult, err := createOrUpdateShortcut(config, shortcutsPath)
		if err != nil {
			r = append(r, results.NewUpdateSteamUserShortcutFailed(config.Name, steamUserId, err.Error()))
			continue
		}

		err = addOrRemoveShortcutTile(config, steamUserId)
		if err != nil {
			r = append(r, results.NewUpdateSteamUserShortcutFailed(config.Name, steamUserId, err.Error()))
			continue
		}

		var ur results.Result

		if len(config.Warnings) == 0 {
			switch fileUpdateResult {
			case shortcuts.UpdatedEntry:
				ur = results.NewUpdateShortcutSuccess(config.Name)
			default:
				ur = results.NewCreateShortcutSuccess(config.Name)
			}
		} else {
			ur = results.NewCreateShortcutSuccessWithWarnings(config.Name,
				strings.Join(config.Warnings, ", "))
		}

		r = append(r, ur)
	}

	return r
}

func createOrUpdateShortcut(config NewShortcutConfig, shortcutsFilePath string) (shortcuts.UpdateResult, error) {
	startDir := path.Dir(config.ExePath)

	onMatch := func(name string, matched *shortcuts.Shortcut) {
		matched.StartDir = startDir
		matched.ExePath = config.ExePath
		matched.LaunchOptions = config.LaunchOptions
		matched.IconPath = config.IconPath
		matched.Tags = config.Tags
	}

	noMatch := func(name string) (shortcuts.Shortcut, bool) {
		return shortcuts.Shortcut{
			AppName:       config.Name,
			ExePath:       config.ExePath,
			StartDir:      startDir,
			IconPath:      config.IconPath,
			LaunchOptions: config.LaunchOptions,
			Tags:          config.Tags,
		}, false
	}

	createOrUpdateConfig := shortcuts.CreateOrUpdateConfig{
		MatchName: config.Name,
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

func addOrRemoveShortcutTile(config NewShortcutConfig, steamUserId string) error {
	tileDetails := grid.ImageDetails{
		DataVerifier:       config.Info.DataLocations,
		OwnerUserId:        steamUserId,
		GameName:           config.Name,
		GameExecutablePath: config.ExePath,
	}

	if len(config.TilePath) == 0 {
		removeConfig := grid.RemoveConfig{
			TargetDetails: tileDetails,
		}

		// TODO: Should this return an error? Perhaps the tile
		//  never existed in the first place?
		err := grid.RemoveImage(removeConfig)
		if err != nil {
			return err
		}

		return nil
	}

	addConfig := grid.AddConfig{
		ImageSourcePath:   config.TilePath,
		ResultDetails:     tileDetails,
		OverwriteExisting: true,
	}

	err := grid.AddImage(addConfig)
	if err != nil {
		return err
	}

	return nil
}

func DeleteShortcut(config DeleteShortcutConfig) []results.Result {
	var r []results.Result

	for steamUserId := range config.Info.IdsToDirPaths {
		shortcutsPath := locations.ShortcutsFilePath(config.Info.DataLocations.RootDirPath(), steamUserId)

		delResult, err := deleteShortcut(config, shortcutsPath)
		if err != nil {
			r = append(r, results.NewDeleteSteamUserShortcutFailure(config.GameName, steamUserId, err.Error()))
			continue
		}

		if !delResult.wasDeleted {
			r = append(r, results.NewDeleteSteamUserShortcutSkipped(config.GameName, steamUserId,
				"no matching shortcut was found"))
			continue
		}

		tileDetails := grid.ImageDetails{
			DataVerifier:       config.Info.DataLocations,
			OwnerUserId:        steamUserId,
			GameExecutablePath: config.LauncherExePath,
			GameName:           config.GameName,
		}

		err = removeShortcutTile(tileDetails)
		if err != nil {
			r = append(r, results.NewDeleteSteamUserShortcutSuccessWarning(config.GameName,
				steamUserId, "failed to delete game tile - " + err.Error()))
			continue
		}

		r = append(r, results.NewDeleteSteamUserShortcutSuccess(config.GameName, steamUserId, ""))
	}

	return r
}

// TODO: This whole operation is IO bound - not good.
//  This needs to be refactored to provide the data set,
//  and then return the modified data set, leaving the
//  IO work to the most minimal amount possible.
func deleteShortcut(config DeleteShortcutConfig, shortcutsFilePath string) (deleteShortcutResult, error) {
	f, err := os.OpenFile(shortcutsFilePath, os.O_RDWR, defaultShortcutsFileMode)
	if err != nil {
		return deleteShortcutResult{}, err
	}
	defer f.Close()

	scs, err := shortcuts.ReadVdfV1(f)
	if err != nil {
		return deleteShortcutResult{}, err
	}

	_, err = f.Seek(0, 0)
	if err != nil {
		return deleteShortcutResult{}, err
	}

	result := deleteShortcutResult{}

	for shortcutIndex := range scs {
		if scs[shortcutIndex].AppName == config.GameName {
			// TODO: There might be multiple entries, so loop over all shortcuts.
			scs = append(scs[:shortcutIndex], scs[shortcutIndex+1:]...)
			result.wasDeleted = true
		}
	}

	err = f.Truncate(0)
	if err != nil {
		return deleteShortcutResult{}, err
	}

	err = shortcuts.WriteVdfV1(scs, f)
	if err != nil {
		return deleteShortcutResult{}, err
	}

	return result, nil
}

func removeShortcutTile(tileDetails grid.ImageDetails) error {
	removeConfig := grid.RemoveConfig{
		TargetDetails: tileDetails,
	}

	err := grid.RemoveImage(removeConfig)
	if err != nil {
		return err
	}

	return nil
}
