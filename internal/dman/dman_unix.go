// +build !windows

package dman

import (
	"errors"
)

type unixManager struct{
	config Config
}

func (o *unixManager) Config() Config {
	return o.config
}

func (o *unixManager) DoManagementCommand(command string) (Status, error) {
	return doManagementCommand(o, command)
}

func (o *unixManager) ReinstallAndStart() error {
	return errors.New("Not implemented for *nix systems")
}

func (o *unixManager) Install() error {
	return errors.New("Not implemented for *nix systems")
}

func (o *unixManager) Remove() error {
	return errors.New("Not implemented for *nix systems")
}

func (o *unixManager) Start() error {
	return errors.New("Not implemented for *nix systems")
}

func (o *unixManager) Stop() error {
	return errors.New("Not implemented for *nix systems")
}

func (o *unixManager) Status() (Status, error) {
	return Unknown, errors.New("Not implemented for *nix systems")
}

func NewManager(config Config) (Manager, error) {
	return &unixManager{
		config: config,
	}, errors.New("Daemon functionalty not currently support on *nix systems")
}
