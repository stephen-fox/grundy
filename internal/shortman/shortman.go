package shortman

import (
	"errors"
	"os"
	"path"
	"strings"

	"github.com/stephen-fox/grundy/internal/settings"
	"github.com/stephen-fox/grundy/internal/steamw"
)

type ShortcutManager interface {
	RefreshAll(steamDataInfo steamw.DataInfo) (CreatedOrUpdated, Deleted)
	Update(gamePaths []string, isDirs bool, steamDataInfo steamw.DataInfo) CreatedOrUpdated
	Delete(gamePaths []string, isDirs bool, steamDataInfo steamw.DataInfo) Deleted
}

type defaultShortcutManager struct {
	config Config
}

func (o *defaultShortcutManager) RefreshAll(steamDataInfo steamw.DataInfo) (CreatedOrUpdated, Deleted) {
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

	c := o.Update(existingDirPaths, true, steamDataInfo)

	d := o.Delete(deletedDirPaths, true, steamDataInfo)

	return c, d
}

func (o *defaultShortcutManager) Update(gamePaths []string, isDirs bool, dataInfo steamw.DataInfo) CreatedOrUpdated {
	c := CreatedOrUpdated{
		gameNamesToResults:    make(map[string]steamw.NewShortcutResult),
		configPathsToLoadErrs: make(map[string]error),
		notAddedToReasons:     make(map[string]string),
		missingIcons:          make(map[string]string),
	}

	for _, gameDir := range gamePaths {
		if strings.HasPrefix(gameDir, o.config.IgnorePathPrefix) {
			continue
		}

		if !isDirs {
			gameDir = path.Dir(gameDir)
		}

		launcherName, hasGameCollection := o.config.App.HasGameCollection(path.Dir(gameDir))
		if !hasGameCollection {
			c.notAddedToReasons[gameDir] = "Game collection does not exist"
			continue
		}

		launcher, hasLauncher := o.config.Launchers.Has(launcherName)
		if !hasLauncher {
			c.notAddedToReasons[gameDir] = "The specified launcher does not " +
				"exist in the launchers settings - '" + launcherName + "'"
			continue
		}

		game := settings.NewGameSettings(gameDir)
		var err error
		if strings.HasSuffix(gameDir, settings.FileExtension) {
			game, err = settings.LoadGameSettings(gameDir, launcher)
		} else {
			exeFilePath, exeExists := game.ExeFullPath(launcher)
			if !exeExists {
				err = errors.New("The executable does not exist - '" + exeFilePath + "'")
			}
		}
		if err != nil {
			c.configPathsToLoadErrs[gameDir] = err
			continue
		}

		added := o.config.KnownGames.AddUniqueGameOnly(game, gameDir)
		if !added {
			c.notAddedToReasons[gameDir] = "The game '" + game.Name() + "' already exists"
			continue
		}

		iconPath, iconExists := game.IconPath()
		if !iconExists {
			c.missingIcons[gameDir] = "Icon does not exist at - '" + iconPath + "'"
		}

		config := steamw.NewShortcutConfig{
			Game:     game,
			Launcher: launcher,
			Info:     dataInfo,
		}

		c.gameNamesToResults[game.Name()] = steamw.CreateOrUpdateShortcutPerId(config)
	}

	return c
}

func (o *defaultShortcutManager) Delete(gamePaths []string, isDirs bool, dataInfo steamw.DataInfo) Deleted {
	var d Deleted

	for _, p := range gamePaths {
		if strings.HasPrefix(p, o.config.IgnorePathPrefix) {
			continue
		}

		if !isDirs {
			p = path.Dir(p)
		}

		gameName, ok := o.config.KnownGames.Disown(p)
		if ok {
			config := steamw.DeleteShortcutConfig{
				GameNames: []string{gameName},
				Info:      dataInfo,
			}

			d.results = append(d.results, steamw.DeleteShortcutPerId(config))
		}
	}

	return d
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
