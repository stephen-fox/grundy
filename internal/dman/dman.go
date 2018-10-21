package dman

import (
	"strings"
	"errors"
)

const (
	name = "grundy"
	description = "Grundy crushes your games into Steam shortcuts " +
		"so you do not have to! Please refer to the usage documentation " +
		"at https://github.com/stephen-fox/grundy for more information."

	install = "install"
	remove  = "remove"
	start   = "start"
	stop    = "stop"
	status  = "status"
)

type Manager interface {
	DoManagementCommand(string) (string, error)
	Install() (string, error)
	Remove() (string, error)
	Start() (string, error)
	Stop() (string, error)
	Status() (string, error)
}

func AvailableManagementCommands() string {
	return "'" + strings.Join([]string{
		install,
		remove,
		start,
		stop,
		status,
	}, "', ") + "'"
}

func doManagementCommand(manager Manager, command string) (string, error) {
	switch strings.ToLower(command) {
	case install:
		return manager.Install()
	case remove:
		return manager.Remove()
	case start:
		return manager.Start()
	case stop:
		return manager.Stop()
	case status:
		return manager.Status()
	}

	return "", errors.New("Unknown daemon management command '" + command + "'")
}
