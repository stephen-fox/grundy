package dman

import "os"

type Config interface {
	Name() string
	Description() string
	ExePath() string
	Arguments() []string
	SetArguments([]string)
	IsAutoStart() bool
	SetAutoStart(bool)
}

type defaultConfig struct {
	exePath     string
	name        string
	description string
	arguments   []string
	autoStart   bool
}

func (o *defaultConfig) Name() string {
	return o.name
}

func (o *defaultConfig) Description() string {
	return o.description
}

func (o *defaultConfig) ExePath() string {
	return o.exePath
}

func (o *defaultConfig) Arguments() []string {
	return o.arguments
}

func (o *defaultConfig) SetArguments(a []string) {
	o.arguments = a
}

func (o *defaultConfig) IsAutoStart() bool {
	return o.autoStart
}

func (o *defaultConfig) SetAutoStart(b bool) {
	o.autoStart = b
}

func NewConfig(name string, description string) (Config, error) {
	exePath, err := os.Executable()
	if err != nil {
		return &defaultConfig{}, err
	}

	return &defaultConfig{
		name:        name,
		description: description,
		exePath:     exePath,
	}, nil
}
