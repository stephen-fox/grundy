package daemonw

import (
	"strings"
)

const (
	daemonStatusPrefix = "Daemon status - "

	Unknown      Status = "unknown"
	Running      Status = "running"
	Stopped      Status = "stopped"
	NotInstalled Status = "not installed"

	GetStatus Command = "status"
	Start     Command = "start"
	Stop      Command = "stop"
	Install   Command = "install"
	Uninstall Command = "uninstall"
)

type Status string

func (o Status) printableStatus() string {
	return daemonStatusPrefix + string(o)
}

func (o Status) string() string {
	return string(o)
}

type Command string

func (o Command) string() string {
	return string(o)
}

type ApplicationLogic interface {
	Start() error
	Stop() error
}

type Config struct {
	Name        string
	Description string
	Username    string
}

func CommandsString() string {
	return "'" + strings.Join(Commands(), "', '") + "'"
}

func Commands() []string {
	return []string{
		GetStatus.string(),
		Start.string(),
		Stop.string(),
		Install.string(),
		Uninstall.string(),
	}
}

func executedCommandMessage(command Command) string {
	return "Executed '" + command.string() + "' daemon control command"
}