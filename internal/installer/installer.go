package installer

import (
	"github.com/stephen-fox/grundy/internal/daemonw"
)

func Install(d daemonw.Daemon) error {
	// Attempt to remove any existing stuff.
	Uninstall(d)

	_, err := d.ExecuteCommand(daemonw.Install)
	if err != nil {
		return err
	}

	status, err := d.Status()
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
		_, err := d.ExecuteCommand(daemonw.Start)
		if err != nil {
			return InstallError{
				reason:            "Failed to start daemon after install - " + err.Error(),
				daemonStartFailed: true,
			}
		}
	}

	return nil
}

func Uninstall(d daemonw.Daemon) error {
	status, statusErr := d.Status()
	if statusErr == nil && status == daemonw.NotInstalled {
		return nil
	}

	if status == daemonw.Running {
		_, err := d.ExecuteCommand(daemonw.Stop)
		if err != nil {
			return UninstallError{
				reason:           "Failed to stop running daemon before uninstall",
				daemonStopFailed: true,
			}
		}
	}

	_, err := d.ExecuteCommand(daemonw.Uninstall)
	if err != nil {
		return UninstallError{
			reason:                err.Error(),
			daemonUninstallFailed: true,
		}
	}

	return nil
}
