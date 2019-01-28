package shortman

import (
	"errors"
	"os"
	"path"
	"strings"

	"github.com/stephen-fox/grundy/internal/results"
	"github.com/stephen-fox/grundy/internal/settings"
	"github.com/stephen-fox/grundy/internal/steamw"
)

type ShortcutManager interface {
	RefreshAll(steamDataInfo steamw.DataInfo) []results.Result
	Update(gamePaths []string, isDirs bool, steamDataInfo steamw.DataInfo) []results.Result
	Delete(gamePaths []string, isDirs bool, steamDataInfo steamw.DataInfo) []results.Result
}

type defaultShortcutManager struct {
	config Config
}

func (o *defaultShortcutManager) RefreshAll(steamDataInfo steamw.DataInfo) []results.Result {
	var deletedDirPaths []string
	var existingDirPaths []string

	for dirPath := range o.config.KnownGames.GameDirPathsToGameNames() {
		info, statErr := os.Stat(dirPath)
		if statErr != nil {
			deletedDirPaths = append(deletedDirPaths, dirPath)
		} else if info.IsDir() {
			existingDirPaths = append(existingDirPaths, dirPath)
		} else {
			deletedDirPaths = append(deletedDirPaths, dirPath)
		}
	}

	var r []results.Result

	r = append(r, o.Update(existingDirPaths, true, steamDataInfo)...)

	r = append(r, o.Delete(deletedDirPaths, true, steamDataInfo)...)

	return r
}

func (o *defaultShortcutManager) Update(gamePaths []string, isDirs bool, dataInfo steamw.DataInfo) []results.Result {
	var r []results.Result

	for _, gameDir := range gamePaths {
		if strings.HasPrefix(gameDir, o.config.IgnorePathPrefix) {
			continue
		}

		if !isDirs {
			gameDir = path.Dir(gameDir)
		}

		collectionName := path.Dir(gameDir)

		launcherName, hasGameCollection := o.config.App.HasGameCollection(collectionName)
		if !hasGameCollection {
			r = append(r, results.NewUpdateShortcutSkipped(gameDir,
				"game collection '" + collectionName + "' does not exist"))
			continue
		}

		launcher, hasLauncher := o.config.Launchers.Has(launcherName)
		if !hasLauncher {
			r = append(r, results.NewUpdateShortcutSkipped(gameDir,
				"the specified launcher does not exist in the launchers settings - '" +
				launcherName + "'"))
			continue
		}

		game := settings.NewGameSettings(gameDir)
		var err error
		if strings.HasSuffix(gameDir, settings.FileExtension) {
			game, err = settings.LoadGameSettings(gameDir, launcher)
		} else {
			exeFilePath, exeExists := game.ExeFullPath(launcher)
			if !exeExists {
				err = errors.New("the game's executable does not exist at '" + exeFilePath + "'")
			}
		}
		if err != nil {
			r = append(r, results.NewUpdateShortcutFailed(gameDir, err.Error()))
			continue
		}

		// TODO: Is this a good idea? Can we be certain that the shortcut
		//  was not removed by someone/thing else besides us?
		added := o.config.KnownGames.AddUniqueGameOnly(game, gameDir)
		if !added {
			r = append(r, results.NewUpdateShortcutSkipped(gameDir, "the game already exists"))
			continue
		}

		var warnings []string

		icon := game.IconPath()
		if !icon.WasDynamicallySelected() && !icon.FileExists() {
			r = append(r, results.NewUpdateShortcutFailed(gameDir, "manual icon does not exist at - '" +
				icon.FilePath() + "'"))
			continue
		} else if icon.WasDynamicallySelected() && !icon.FileExists() {
			warnings = append(warnings, "no icon was provided")
		}

		tile := game.TilePath()
		if !tile.WasDynamicallySelected() && !tile.FileExists() {
			r = append(r, results.NewUpdateShortcutFailed(gameDir, "manual tile does not exist at - '" +
				tile.FilePath() + "'"))
			continue
		} else if tile.WasDynamicallySelected() && !tile.FileExists() {
			warnings = append(warnings, "no tile was provided")
		}

		config := steamw.NewShortcutConfig{
			Name:          game.Name(),
			LaunchOptions: createLauncherArgs(game, launcher),
			ExePath:       launcher.ExePath(),
			IconPath:      icon.FilePath(),
			TilePath:      tile.FilePath(),
			Tags:          game.Categories(),
			Info:          dataInfo,
			Warnings:      warnings,
		}

		r = append(r, steamw.CreateOrUpdateShortcut(config)...)
	}

	return r
}

// TODO: Refactor this.
func createLauncherArgs(game settings.GameSettings, launcher settings.Launcher) []string {
	var options []string

	if game.ShouldOverrideLauncherArgs() {
		options = append(options, game.LauncherOverrideArgs())
	} else {
		if len(launcher.DefaultArgs()) > 0 {
			options = append(options, launcher.DefaultArgs())
		}

		if len(game.AdditionalLauncherArgs()) > 0 {
			options = append(options, game.AdditionalLauncherArgs())
		}
	}

	exePath, _ := game.ExeFullPath(launcher)

	options = append(options, "\"" + exePath + "\"")

	return options
}

func (o *defaultShortcutManager) Delete(gamePaths []string, isDirs bool, dataInfo steamw.DataInfo) []results.Result {
	var r []results.Result

	for _, p := range gamePaths {
		if strings.HasPrefix(p, o.config.IgnorePathPrefix) {
			continue
		}

		if !isDirs {
			p = path.Dir(p)
		}

		var launcherExePath string

		// Do not delete if there is an executable in the directory.
		launcherName, hasCollection := o.config.App.HasGameCollection(p)
		if hasCollection {
			launcher, hasLauncher := o.config.Launchers.Has(launcherName)
			if hasLauncher {
				launcherExePath = launcher.ExePath()
				game := settings.NewGameSettings(p)
				exePath, exeExists := game.ExeFullPath(launcher)
				if exeExists {
					r = append(r, results.NewDeleteShortcutSkipped(game.Name(),
						"a game executable still exists in its directory at '" + exePath + "'"))
					continue
				}
			}
		}

		gameName, ok := o.config.KnownGames.Disown(p)
		if ok {
			config := steamw.DeleteShortcutConfig{
				GameName:        gameName,
				Info:            dataInfo,
				SkipTileDelete:  len(launcherExePath) == 0,
				LauncherExePath: launcherExePath,
			}

			r = append(r, steamw.DeleteShortcut(config)...)
		}
	}

	return r
}

type Config struct {
	App              settings.AppSettings
	KnownGames       settings.KnownGamesSettings
	Launchers        settings.LaunchersSettings
	IgnorePathPrefix string
}

func NewShortcutManager(config Config) ShortcutManager {
	return &defaultShortcutManager{
		config: config,
	}
}
