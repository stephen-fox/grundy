package dman

import (
	"errors"

	"github.com/takama/daemon"
	"github.com/stephen-fox/userutil"
)

type windowsManager struct{
	d daemon.Daemon
}

func (o *windowsManager) DoManagementCommand(command string) (string, error) {
	return doManagementCommand(o, command)
}

func (o *windowsManager) Install() (string, error) {
	return o.d.Install()
}

func (o *windowsManager) Remove() (string, error) {
	return o.d.Remove()
}

func (o *windowsManager) Start() (string, error) {
	return o.d.Start()
}

func (o *windowsManager) Stop() (string, error) {
	return o.d.Stop()
}

func (o *windowsManager) Status() (string, error) {
	return o.d.Status()
}

func NewManager() (Manager, error) {
	err := userutil.IsRoot()
	if err != nil {
		return &windowsManager{}, errors.New("Application must be run as Administrator to manage the daemon")
	}

	d, err := daemon.New(name, description)
	if err != nil {
		return &windowsManager{}, err
	}

	return &windowsManager{
		d: d,
	}, nil
}
