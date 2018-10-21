package dman

import (
	"strings"
	"errors"
)

const (
	install           = "install"
	reinstallAndStart = "reinstall_and_start"
	remove            = "remove"
	start             = "start"
	stop              = "stop"
	status            = "status"
)

const (
	Stopping Status = "stopping"
	Stopped  Status = "stopped"
	Starting Status = "starting"
	Running  Status = "running"
	Resuming Status = "resuming"
	Pausing  Status = "pausing"
	Paused   Status = "paused"
	Unknown  Status = "unknown"
)

type Status string

type Manager interface {
	Config() Config
	DoManagementCommand(string) (Status, error)
	ReinstallAndStart() error
	Install() error
	Remove() error
	Start() error
	Stop() error
	Status() (Status, error)
}

func AvailableManagementCommands() string {
	return "'" + strings.Join([]string{
		install,
		reinstallAndStart,
		remove,
		start,
		stop,
		status,
	}, "', ") + "'"
}

func doManagementCommand(manager Manager, command string) (Status, error) {
	var err error

	switch strings.ToLower(command) {
	case install:
		err = manager.Install()
	case reinstallAndStart:
		err = manager.ReinstallAndStart()
	case remove:
		err = manager.Remove()
	case start:
		err = manager.Start()
	case stop:
		err = manager.Stop()
	case status:
		break
	default:
		return Unknown, errors.New("Unknown daemon management command '" + command + "'")
	}

	if err != nil {
		return Unknown, err
	}

	return manager.Status()
}
