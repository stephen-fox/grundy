package daemonw

import "strings"

const (
	runningStatus      = daemonStatusPrefix + "running"
	stoppedStatus      = daemonStatusPrefix + "stopped"
	unknownStatus      = daemonStatusPrefix + "unknown"
	daemonStatusPrefix = "Daemon status - "
)

const (
	status    Command = "status"
	start     Command = "start"
	stop      Command = "stop"
	install   Command = "install"
	uninstall Command = "uninstall"
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
		status.string(),
		start.string(),
		stop.string(),
		install.string(),
		uninstall.string(),
	}
}

func executedCommandMessage(command Command) string {
	return "Executed '" + command.string() + "' daemon control command"
}
