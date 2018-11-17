package installer

import (
	"github.com/stephen-fox/grundy/internal/daemonw"
)

func Install(config daemonw.Config) error {
	// Attempt to remove any existing stuff.
	Uninstall(config)

	_, err := daemonw.ExecuteCommand(daemonw.Install, config)
	if err != nil {
		return err
	}

	status, err := daemonw.CurrentStatus(config)
	if err != nil {
		return InstallError{
			reason:         "Failed to get daemon status after installation - " + err.Error(),
			noDaemonStatus: true,
		}
	}

	switch status {
	case daemonw.NotInstalled:
		return InstallError{
			reason:                     "Daemon installation failed for unknown reason",
			daemonInstallFailedUnknown: true,
		}
	case daemonw.Stopped:
		_, err := daemonw.ExecuteCommand(daemonw.Start, config)
		if err != nil {
			return InstallError{
				reason:            "Failed to start daemon after install - " + err.Error(),
				daemonStartFailed: true,
			}
		}
	}

	return nil
}

func Uninstall(config daemonw.Config) error {
	status, statusErr := daemonw.CurrentStatus(config)
	if statusErr == nil && status == daemonw.NotInstalled {
		return nil
	}

	if status == daemonw.Running {
		_, err := daemonw.ExecuteCommand(daemonw.Stop, config)
		if err != nil {
			return UninstallError{
				reason:           "Failed to stop running daemon before uninstall",
				daemonStopFailed: true,
			}
		}
	}

	_, err := daemonw.ExecuteCommand(daemonw.Uninstall, config)
	if err != nil {
		return UninstallError{
			reason:                err.Error(),
			daemonUninstallFailed: true,
		}
	}

	return nil
}
