// +build !windows

package dman

import (
	"github.com/takama/daemon"
	"github.com/stephen-fox/userutil"
)

type unixManager struct{
	d daemon.Daemon
}

func (o *unixManager) DoManagementCommand(command string) (string, error) {
	return doManagementCommand(o, command)
}

func (o *unixManager) Install() (string, error) {
	return o.d.Install()
}

func (o *unixManager) Remove() (string, error) {
	return o.d.Remove()
}

func (o *unixManager) Start() (string, error) {
	return o.d.Start()
}

func (o *unixManager) Stop() (string, error) {
	return o.d.Stop()
}

func (o *unixManager) Status() (string, error) {
	return o.d.Status()
}

func NewManager() (Manager, error) {
	err := userutil.IsRoot()
	if err != nil {
		return &unixManager{}, err
	}

	d, err := daemon.New(name, description)
	if err != nil {
		return &unixManager{}, err
	}

	return &unixManager{
		d: d,
	}, nil
}
