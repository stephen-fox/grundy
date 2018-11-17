package daemonw

import "strings"

const (
	runningStatus      = daemonStatusPrefix + "running"
	stoppedStatus      = daemonStatusPrefix + "stopped"
	unknownStatus      = daemonStatusPrefix + "unknown"
	notInstalledStatus = daemonStatusPrefix + "not installed"
	daemonStatusPrefix = "Daemon status - "
)

const (
	Status    Command = "status"
	Start     Command = "start"
	Stop      Command = "stop"
	Install   Command = "install"
	Uninstall Command = "uninstall"
)

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
		Status.string(),
		Start.string(),
		Stop.string(),
		Install.string(),
		Uninstall.string(),
	}
}

func executedCommandMessage(command Command) string {
	return "Executed '" + command.string() + "' daemon control command"
}
