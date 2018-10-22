package settings

import (
	"io"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

const (
	FileExtension = ".grundy.ini"
	ExampleSuffix = "-example"

	defaultFileMode = 0644

	managedFileComment = "[WARNING] DO NOT EDIT THIS FILE. This file is managed by the application."

	none          section = ""
	appSettings   section = "settings"
	appWatchPaths section = "watch_paths"

	appAutoStart key = "auto_start"

	launcherExePath     key = "exe_path"
	launcherDefaultArgs key = "default_args"

	gameName           key = "name"
	gameExePath        key = "exe_path"
	gameLauncher       key = "launcher"
	gameOverrideArgs   key = "override_args"
	gameAdditionalArgs key = "additional_args"
	gameIcon           key = "icon"
	gameCategories     key = "categories_comma_separated"

	gameCategoriesSeparator = ","
)

type section string

type key string

type SaveableSettings interface {
	Filename(additionalSuffix string) string
	Reload(filePath string) error
	Save(io.Writer) error
	ResetToDefaults()
}

type AppSettings interface {
	SaveableSettings
	IsAutoStart() bool
	SetAutoStart(bool)
	WatchPaths() []string
	AddWatchPath(string)
	RemoveWatchPath(string)
	HasWatchPath(string) bool
}

type defaultAppSettings struct {
	config configFile
}

func (o *defaultAppSettings) Filename(additionalSuffix string) string {
	return "app" + additionalSuffix + FileExtension
}

func (o *defaultAppSettings) Reload(filePath string) error {
	return o.config.Reload(filePath)
}

func (o *defaultAppSettings) Save(w io.Writer) error {
	return o.config.Save(w)
}

func (o *defaultAppSettings) ResetToDefaults() {
	o.config.Clear()

	o.config.AddSection(appSettings)
	o.config.AddOrUpdateKeyValue(appSettings, appAutoStart, strconv.FormatBool(true))
	o.config.DeleteSection(appWatchPaths)
	o.config.AddSection(appWatchPaths)
}

func (o *defaultAppSettings) IsAutoStart() bool {
	v, err := strconv.ParseBool(o.config.KeyValue(appSettings, appAutoStart))
	if err != nil {
		return false
	}

	return v
}

func (o *defaultAppSettings) SetAutoStart(v bool) {
	o.config.AddOrUpdateKeyValue(appSettings, appAutoStart, strconv.FormatBool(v))
}

func (o *defaultAppSettings) WatchPaths() []string {
	return o.config.SectionKeys(appWatchPaths)
}

func (o *defaultAppSettings) AddWatchPath(p string) {
	o.config.AddValueToSection(appWatchPaths, p)
}

func (o *defaultAppSettings) RemoveWatchPath(p string) {
	o.config.DeleteKey(appWatchPaths, key(p))
}

func (o *defaultAppSettings) HasWatchPath(dirPath string) bool {
	return o.config.HasKey(appWatchPaths, key(dirPath))
}

type LaunchersSettings interface {
	SaveableSettings
	Has(name string) (Launcher, bool)
	AddOrUpdate(Launcher)
	Remove(Launcher)
}

type defaultLaunchersSettings struct {
	config configFile
}

func (o *defaultLaunchersSettings) Filename(additionalSuffix string) string {
	return "launchers" + additionalSuffix + FileExtension
}

func (o *defaultLaunchersSettings) Reload(filePath string) error {
	return o.config.Reload(filePath)
}

func (o *defaultLaunchersSettings) Save(w io.Writer) error {
	return o.config.Save(w)
}

func (o *defaultLaunchersSettings) ResetToDefaults() {
	o.config.Clear()

	l := NewLauncher()

	o.config.AddOrUpdateKeyValue(section(l.Name()), launcherExePath, l.ExePath())
	o.config.AddOrUpdateKeyValue(section(l.Name()), launcherDefaultArgs, l.DefaultArgs())
}

func (o *defaultLaunchersSettings) Has(name string) (Launcher, bool) {
	l := NewLauncher()

	sec := section(name)

	if o.config.HasSection(sec) {
		l.SetName(name)
		l.SetExePath(o.config.KeyValue(sec, launcherExePath))
		l.SetDefaultArgs(o.config.KeyValue(sec, launcherDefaultArgs))

		return l, true
	}

	return l, false
}

func (o *defaultLaunchersSettings) AddOrUpdate(l Launcher) {
	sec := section(l.Name())

	o.config.AddOrUpdateKeyValue(sec, launcherExePath, l.ExePath())
	o.config.AddOrUpdateKeyValue(sec, launcherDefaultArgs, l.DefaultArgs())
}

func (o *defaultLaunchersSettings) Remove(l Launcher) {
	o.config.DeleteSection(section(l.Name()))
}

type Launcher interface {
	ResetToDefaults()
	SetName(string)
	Name() string
	SetExePath(string)
	ExePath() string
	ExeDirPath() string
	SetDefaultArgs(string)
	DefaultArgs() string
}

type defaultLauncherSettings struct {
	name        string
	exePath     string
	defaultArgs string
}

func (o *defaultLauncherSettings) ResetToDefaults() {
	o.name = "example-launcher"

	if runtime.GOOS == "windows" {
		o.exePath = "C:\\path\\to\\launcher\\executable.file"
	} else {
		o.exePath = "/path/to/launcher/executable.file"
	}

	o.defaultArgs = ""
}

func (o *defaultLauncherSettings) SetName(name string) {
	o.name = name
}

func (o *defaultLauncherSettings) Name() string {
	return o.name
}

func (o *defaultLauncherSettings) SetExePath(filePath string) {
	o.exePath = filePath
}

func (o *defaultLauncherSettings) ExePath() string {
	return o.exePath
}

func (o *defaultLauncherSettings) ExeDirPath() string {
	return path.Dir(o.ExePath())
}

func (o *defaultLauncherSettings) SetDefaultArgs(args string) {
	o.defaultArgs = args
}

func (o *defaultLauncherSettings) DefaultArgs() string {
	return o.defaultArgs
}

type GameSettings interface {
	SaveableSettings
	SetName(string)
	Name() string
	SetExePath(string)
	ExePath(relativeToSettings bool) string
	SetLauncher(string)
	Launcher() string
	ShouldOverrideLauncherArgs() bool
	SetLauncherOverrideArgs(string)
	LauncherOverrideArgs() string
	SetAdditionalLauncherArgs(string)
	AdditionalLauncherArgs() string
	SetIconPath(string)
	IconPath() string
	AddCategory(string)
	RemoveCategory(string)
	SetCategories([]string)
	Categories() []string
}

type defaultGameSettings struct {
	config   configFile
	filePath string
}

func (o *defaultGameSettings) Filename(additionalSuffix string) string {
	return "game" + additionalSuffix + FileExtension
}

func (o *defaultGameSettings) Reload(filePath string) error {
	return o.config.Reload(filePath)
}

func (o *defaultGameSettings) ResetToDefaults() {
	o.config.Clear()

	o.config.AddOrUpdateKeyValue(none, gameName, "example-game")
	o.config.AddOrUpdateKeyValue(none, gameExePath, "example.exe")
	o.config.AddOrUpdateKeyValue(none, gameLauncher, "example-launcher")
	o.config.AddOrUpdateKeyValue(none, gameAdditionalArgs, "")
	o.config.AddOrUpdateKeyValue(none, gameOverrideArgs, "")
	if runtime.GOOS == "windows" {
		o.config.AddOrUpdateKeyValue(none, gameIcon, "C:\\path\\to\\game-icon.png")
	} else {
		o.config.AddOrUpdateKeyValue(none, gameIcon, "/path/to/game-icon.png")
	}
	o.config.AddOrUpdateKeyValue(none, gameCategories, "My Cool Category,Another Cool Category,some-other category")
}

func (o *defaultGameSettings) Save(w io.Writer) error {
	return o.config.Save(w)
}

func (o *defaultGameSettings) SetName(name string) {
	o.config.AddOrUpdateKeyValue(none, gameName, name)
}

func (o *defaultGameSettings) Name() string {
	return o.config.KeyValue(none, gameName)
}

func (o *defaultGameSettings) SetExePath(p string) {
	o.config.AddOrUpdateKeyValue(none, gameExePath, appendDoubleQuotesIfNeeded(p))
}

func (o *defaultGameSettings) ExePath(relativeToSettings bool) string {
	exePath := o.config.KeyValue(none, gameExePath)

	if relativeToSettings {
		exePath = path.Join(path.Dir(o.filePath), exePath)
	}

	return appendDoubleQuotesIfNeeded(exePath)
}

func (o *defaultGameSettings) SetLauncher(name string) {
	o.config.AddOrUpdateKeyValue(section(""), gameLauncher, name)
}

func (o *defaultGameSettings) Launcher() string {
	return o.config.KeyValue(none, gameLauncher)
}

func (o *defaultGameSettings) ShouldOverrideLauncherArgs() bool {
	return len(o.LauncherOverrideArgs()) > 0
}

func (o *defaultGameSettings) SetLauncherOverrideArgs(args string) {
	o.config.AddOrUpdateKeyValue(none, gameOverrideArgs, args)
}

func (o *defaultGameSettings) LauncherOverrideArgs() string {
	return o.config.KeyValue(none, gameOverrideArgs)
}

func (o *defaultGameSettings) SetAdditionalLauncherArgs(args string) {
	o.config.AddOrUpdateKeyValue(none, gameAdditionalArgs, args)
}

func (o *defaultGameSettings) AdditionalLauncherArgs() string {
	return o.config.KeyValue(none, gameAdditionalArgs)
}

func (o *defaultGameSettings) SetIconPath(iconPath string) {
	o.config.AddOrUpdateKeyValue(none, gameIcon, iconPath)
}

func (o *defaultGameSettings) IconPath() string {
	return o.config.KeyValue(none, gameIcon)
}

func (o *defaultGameSettings) AddCategory(c string) {
	current := o.Categories()

	for _, s := range current {
		if s == c {
			return
		}
	}

	current = append(current, c)
	o.SetCategories(current)
}

func (o *defaultGameSettings) RemoveCategory(c string) {
	current := o.Categories()

	for i, s := range current {
		if s == c {
			current = append(current[:i], current[i+1:]...)
			o.SetCategories(current)
			return
		}
	}
}

func (o *defaultGameSettings) SetCategories(cats []string) {
	o.config.AddOrUpdateKeyValue(none, gameCategories, strings.Join(cats, gameCategoriesSeparator))
}

func (o *defaultGameSettings) Categories() []string {
	data := o.config.KeyValue(none, gameCategories)

	if len(data) == 0 {
		return []string{}
	}

	return strings.Split(data, gameCategoriesSeparator)
}

type KnownGamesSettings interface {
	SaveableSettings
	AddUniqueGameOnly(game GameSettings, configFilePath string) bool
	Remove(configFilePath string) (gameName string, ok bool)
	RemoveNonExistingConfigs() (filePathsToGameNames map[string]string)
	Has(configFilePath string) (gameName string, ok bool)
}

type defaultKnownGamesSettings struct {
	mutex    *sync.Mutex
	config   configFile
	filePath string
}

func (o *defaultKnownGamesSettings) Filename(additionalSuffix string) string {
	return ".known-games" + additionalSuffix + FileExtension
}

func (o *defaultKnownGamesSettings) Reload(filePath string) error {
	return o.config.Reload(filePath)
}

func (o *defaultKnownGamesSettings) ResetToDefaults() {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	o.config.Clear()
	o.config.SetSectionComment(none, managedFileComment)
}

func (o *defaultKnownGamesSettings) Save(w io.Writer) error {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	o.config.SetSectionComment(none, managedFileComment)

	return o.config.Save(w)
}

func (o *defaultKnownGamesSettings) AddUniqueGameOnly(game GameSettings, filePath string) bool {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	var count int

	for _, gameName := range o.config.SectionValues(none) {
		if gameName == game.Name() {
			count++

			if count > 1 {
				return false
			}
		}
	}

	o.config.AddOrUpdateKeyValue(none, key(filePath), game.Name())
	o.saveUnsafe()

	return true
}

func (o *defaultKnownGamesSettings) Remove(filePath string) (string, bool) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	return o.removeUnsafe(filePath)
}

func (o *defaultKnownGamesSettings) RemoveNonExistingConfigs() map[string]string {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	removedFilePathsToGameNames := make(map[string]string)

	for _, filePath := range o.config.SectionKeys(none) {
		_, statErr := os.Stat(filePath)
		if statErr == nil {
			continue
		}

		name, ok := o.removeUnsafe(filePath)
		if ok {
			removedFilePathsToGameNames[filePath] = name
		}
	}

	return removedFilePathsToGameNames
}

func (o *defaultKnownGamesSettings) removeUnsafe(filePath string) (string, bool) {
	if o.config.HasKey(none, key(filePath)) {
		gameName := o.config.KeyValue(none, key(filePath))
		o.config.DeleteKey(none, key(filePath))
		o.saveUnsafe()

		return gameName, true
	}

	return "", false
}

func (o *defaultKnownGamesSettings) Has(filePath string) (string, bool) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if o.config.HasKey(none, key(filePath)) {
		gameName := o.config.KeyValue(none, key(filePath))

		return gameName, true
	}

	return "", false
}

func (o *defaultKnownGamesSettings) saveUnsafe() error {
	f, err := os.OpenFile(o.filePath, os.O_WRONLY|os.O_CREATE, defaultFileMode)
	if err != nil {
		return err
	}
	defer f.Close()

	err = f.Truncate(0)
	if err != nil {
		return err
	}

	err = o.config.Save(f)
	if err != nil {
		return err
	}

	return nil
}

func NewAppSettings() AppSettings {
	s := &defaultAppSettings{
		config: newEmptyIniFile(),
	}

	s.ResetToDefaults()

	return s
}

func NewLaunchersSettings() LaunchersSettings {
	s := &defaultLaunchersSettings{
		config: newEmptyIniFile(),
	}

	s.ResetToDefaults()

	return s
}

func NewLauncher() Launcher {
	s := &defaultLauncherSettings{}

	s.ResetToDefaults()

	return s
}

func NewGameSettings() GameSettings {
	s := &defaultGameSettings{
		config: newEmptyIniFile(),
	}

	s.ResetToDefaults()

	return s
}

func LoadOrCreateKnownGamesSettings(parentDirPath string) (KnownGamesSettings, bool) {
	s := &defaultKnownGamesSettings{
		config: newEmptyIniFile(),
		mutex:  &sync.Mutex{},
	}

	s.ResetToDefaults()

	s.filePath = path.Join(parentDirPath, s.Filename(""))

	f, err := loadIniConfigFile(s.filePath)
	if err != nil {
		s.saveUnsafe()

		return s, false
	}

	s.config = f

	return s, true
}

func LoadAppSettings(filePath string) (AppSettings, error) {
	f, err := loadIniConfigFile(filePath)
	if err != nil {
		return &defaultAppSettings{}, err
	}

	return &defaultAppSettings{
		config: f,
	}, nil
}

func LoadLaunchersSettings(filePath string) (LaunchersSettings, error) {
	f, err := loadIniConfigFile(filePath)
	if err != nil {
		return &defaultLaunchersSettings{}, err
	}

	return &defaultLaunchersSettings{
		config: f,
	}, nil
}

func LoadGameSettings(filePath string) (GameSettings, error) {
	f, err := loadIniConfigFile(filePath)
	if err != nil {
		return &defaultGameSettings{}, err
	}

	return &defaultGameSettings{
		config:   f,
		filePath: filePath,
	}, nil
}

func Create(parentDirPath string, filenameSuffix string, s SaveableSettings) error {
	err := CreateDir(parentDirPath)
	if err != nil {
		return err
	}

	filePath := path.Join(parentDirPath, s.Filename(filenameSuffix))

	f, err := os.OpenFile(filePath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, defaultFileMode)
	if err != nil {
		return err
	}

	err = s.Save(f)
	if err != nil {
		return err
	}
	defer f.Close()

	return nil
}

func appendDoubleQuotesIfNeeded(s string) string {
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
