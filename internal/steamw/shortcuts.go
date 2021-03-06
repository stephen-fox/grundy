package steamw

import (
	"bytes"
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
	LaunchOptions []string
	ExePath       string
	IconPath      string
	GridImagePath string
	Tags          []string
	Info          DataInfo
	Warnings      []string
	startDir      string
}

func (o *NewShortcutConfig) clean() {
	o.ExePath = doubleQuoteIfNeeded(o.ExePath)
	o.IconPath = doubleQuoteIfNeeded(o.IconPath)
	o.startDir = doubleQuoteIfNeeded(path.Dir(o.ExePath))
}

// TODO: Clean?
type DeleteShortcutConfig struct {
	SkipGridImageDelete bool
	LauncherExePath     string
	GameName            string
	Info                DataInfo
}

type deleteShortcutResult struct {
	wasDeleted bool
}

func CreateOrUpdateShortcut(config NewShortcutConfig) []results.Result {
	config.clean()

	var r []results.Result

	for steamUserId := range config.Info.IdsToDirPaths {
		shortcutsPath := locations.ShortcutsFilePath(config.Info.DataLocations.RootDirPath(), steamUserId)

		fileUpdateResult, err := createOrUpdateShortcut(config, shortcutsPath)
		if err != nil {
			r = append(r, results.NewUpdateSteamUserShortcutFailed(config.Name, steamUserId, err.Error()))
			continue
		}

		err = addOrRemoveShortcutGridImage(config, steamUserId)
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
	onMatch := func(name string, matched *shortcuts.Shortcut) {
		matched.StartDir = config.startDir
		matched.ExePath = config.ExePath
		matched.LaunchOptions = launchOptionsSliceToString(config.LaunchOptions)
		matched.IconPath = config.IconPath
		matched.Tags = config.Tags
	}

	noMatch := func(name string) (shortcuts.Shortcut, bool) {
		return shortcuts.Shortcut{
			AppName:       config.Name,
			ExePath:       config.ExePath,
			StartDir:      config.startDir,
			IconPath:      config.IconPath,
			LaunchOptions: launchOptionsSliceToString(config.LaunchOptions),
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

func addOrRemoveShortcutGridImage(config NewShortcutConfig, steamUserId string) error {
	imageDetails := grid.ImageDetails{
		DataVerifier:       config.Info.DataLocations,
		OwnerUserId:        steamUserId,
		GameName:           config.Name,
		GameExecutablePath: config.ExePath,
	}

	if len(config.GridImagePath) == 0 {
		removeConfig := grid.RemoveConfig{
			TargetDetails: imageDetails,
		}

		// TODO: Should this return an error? Perhaps the grid
		//  image never existed in the first place?
		err := grid.RemoveImage(removeConfig)
		if err != nil {
			return err
		}

		return nil
	}

	gridAddConfig := grid.AddConfig{
		ImageSourcePath:   config.GridImagePath,
		ResultDetails:     imageDetails,
		OverwriteExisting: true,
	}

	err := grid.AddImage(gridAddConfig)
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

		imageDetails := grid.ImageDetails{
			DataVerifier:       config.Info.DataLocations,
			OwnerUserId:        steamUserId,
			GameExecutablePath: config.LauncherExePath,
			GameName:           config.GameName,
		}

		err = removeShortcutGridImage(imageDetails)
		if err != nil {
			r = append(r, results.NewDeleteSteamUserShortcutSuccessWarning(config.GameName,
				steamUserId, "failed to delete game grid image - " + err.Error()))
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

	currentShortcuts, err := shortcuts.ReadVdfV1(f)
	if err != nil {
		return deleteShortcutResult{}, err
	}

	_, err = f.Seek(0, 0)
	if err != nil {
		return deleteShortcutResult{}, err
	}

	result := deleteShortcutResult{}

	// Based on work by "tomasz":
	// https://stackoverflow.com/a/20551116
	i := 0
	for _, sc := range currentShortcuts {
		if sc.AppName == config.GameName {
			result.wasDeleted = true
			continue
		}
		currentShortcuts[i] = sc
		i++
	}
	currentShortcuts = currentShortcuts[:i]

	err = f.Truncate(0)
	if err != nil {
		return deleteShortcutResult{}, err
	}

	err = shortcuts.WriteVdfV1(currentShortcuts, f)
	if err != nil {
		return deleteShortcutResult{}, err
	}

	return result, nil
}

func removeShortcutGridImage(imageDetails grid.ImageDetails) error {
	removeConfig := grid.RemoveConfig{
		TargetDetails: imageDetails,
	}

	err := grid.RemoveImage(removeConfig)
	if err != nil {
		return err
	}

	return nil
}

func launchOptionsSliceToString(options []string) string {
	buffer := bytes.NewBuffer(nil)

	for i := range options {
		buffer.WriteString(options[i])
		buffer.WriteString(" ")
	}

	return buffer.String()
}

func doubleQuoteIfNeeded(s string) string {
	if strings.Contains(s, " ") {
		doubleQuote := "\""

		if !strings.HasPrefix(s, doubleQuote) {
			s = doubleQuote + s
		}

		if !strings.HasSuffix(s, doubleQuote) {
			s = s + doubleQuote
		}
	}

	return s
}
