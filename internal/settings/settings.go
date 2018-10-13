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

	defaultDirMode  = 0700
	defaultFileMode = 0600

	none          section = ""
	appSettings   section = "settings"
	appWatchPaths section = "watch_paths"

	appAutoStart key = "auto_start"

	launcherExe         key = "exe"
	launcherDefaultArgs key = "default_args"

	gameName           key = "name"
	gameLauncher       key = "launcher"
	gameOverrideArgs   key = "override_args"
	gameAdditionalArgs key = "additional_args"
	gameIcon           key = "icon"
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
	AddOrUpdate(Launcher)
	Remove(Launcher)
}

type defaultLaunchersSettings struct {
	mutex  *sync.Mutex
	config configFile
}

func (o *defaultLaunchersSettings) Filename(additionalSuffix string) string {
	return "launchers" + additionalSuffix + FileExtension
}

func (o *defaultLaunchersSettings) Reload(filePath string) error {
	return o.config.Reload(filePath)
}

func (o *defaultLaunchersSettings) Save(w io.Writer) error {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	return o.config.Save(w)
}

func (o *defaultLaunchersSettings) ResetToDefaults() {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	o.config.Clear()

	l := NewLauncher()

	o.config.AddOrUpdateKeyValue(section(l.Name()), launcherExe, l.ExePath())
	o.config.AddOrUpdateKeyValue(section(l.Name()), launcherDefaultArgs, l.DefaultArgs())
}

func (o *defaultLaunchersSettings) AddOrUpdate(l Launcher) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	o.config.AddOrUpdateKeyValue(section(l.Name()), launcherExe, l.ExePath())
	o.config.AddOrUpdateKeyValue(section(l.Name()), launcherDefaultArgs, l.DefaultArgs())
}

func (o *defaultLaunchersSettings) Remove(l Launcher) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	o.config.DeleteSection(section(l.Name()))
}

type Launcher interface {
	ResetToDefaults()
	SetName(string)
	Name() string
	SetExePath(string)
	ExePath() string
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
	SetLauncher(string)
	Launcher() string
	SetLauncherOverrideArgs(string)
	LauncherOverrideArgs() string
	SetAdditionalLauncherArgs(string)
	AdditionalLauncherArgs() string
	SetIcon(string)
	Icon() string
}

type defaultGameSettings struct {
	config configFile
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
	o.config.AddOrUpdateKeyValue(none, gameLauncher, "example-launcher")
	o.config.AddOrUpdateKeyValue(none, gameAdditionalArgs, "")
	o.config.AddOrUpdateKeyValue(none, gameOverrideArgs, "")
	if runtime.GOOS == "windows" {
		o.config.AddOrUpdateKeyValue(none, gameIcon, "C:\\path\\to\\game-icon.png")
	} else {
		o.config.AddOrUpdateKeyValue(none, gameIcon, "/path/to/game-icon.png")
	}
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

func (o *defaultGameSettings) SetLauncher(name string) {
	o.config.AddOrUpdateKeyValue(section(""), gameLauncher, name)
}

func (o *defaultGameSettings) Launcher() string {
	return o.config.KeyValue(none, gameLauncher)
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

func (o *defaultGameSettings) SetIcon(iconPath string) {
	o.config.AddOrUpdateKeyValue(none, gameIcon, iconPath)
}

func (o *defaultGameSettings) Icon() string {
	return o.config.KeyValue(none, gameIcon)
}

type KnownGamesSettings interface {
	SaveableSettings
	Add(dirPath string)
	Remove(dirPath string)
	Has(dirPath string) bool
}

type defaultKnownGamesSettings struct {
	mutex        *sync.Mutex
	config       configFile
	gameDirPaths []string
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

	o.gameDirPaths = []string{}
	o.config.Clear()
}

func (o *defaultKnownGamesSettings) Save(w io.Writer) error {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	for _, p := range o.gameDirPaths {
		o.config.AddValueToSection(none, p)
	}

	return o.config.Save(w)
}

func (o *defaultKnownGamesSettings) Add(dirPath string) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	o.gameDirPaths = append(o.gameDirPaths, dirPath)
}

func (o *defaultKnownGamesSettings) Remove(dirPath string) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	for i := range o.gameDirPaths {
		if o.gameDirPaths[i] == dirPath {
			o.gameDirPaths = append(o.gameDirPaths[:i], o.gameDirPaths[i+1:]...)
		}
	}
}

func (o *defaultKnownGamesSettings) Has(dirPath string) bool {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	for _, d := range o.gameDirPaths {
		if d == dirPath {
			return true
		}
	}

	return false
}

func DirPath() string {
	var parentPath string

	switch runtime.GOOS {
	case "darwin":
		fallthrough
	case "linux":
		parentPath = os.Getenv("HOME")
	case "windows":
		parentPath = strings.Replace(os.Getenv("USERPROFILE"), "\\", "/", -1)
	}

	return path.Join(parentPath, ".grundy")
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
		mutex:  &sync.Mutex{},
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

func NewKnownGamesSettings() KnownGamesSettings {
	return &defaultKnownGamesSettings{
		config: newEmptyIniFile(),
		mutex:  &sync.Mutex{},
	}
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
		mutex:  &sync.Mutex{},
		config: f,
	}, nil
}

func LoadGameSettings(filePath string) (GameSettings, error) {
	f, err := loadIniConfigFile(filePath)
	if err != nil {
		return &defaultGameSettings{}, err
	}

	return &defaultGameSettings{
		config: f,
	}, nil
}

func LoadKnownGames(filePath string) (KnownGamesSettings, error) {
	f, err := loadIniConfigFile(filePath)
	if err != nil {
		return &defaultKnownGamesSettings{}, err
	}

	return &defaultKnownGamesSettings{
		mutex:  &sync.Mutex{},
		config: f,
	}, nil
}

func Create(parentDirPath string, filenameSuffix string, s SaveableSettings) error {
	err := os.MkdirAll(parentDirPath, defaultDirMode)
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

