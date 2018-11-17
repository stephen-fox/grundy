package installer

import (
	"github.com/stephen-fox/grundy/internal/cyberdaemon"
)

func Install(d cyberdaemon.Daemon) error {
	// Attempt to remove any existing stuff.
	Uninstall(d)

	_, err := d.ExecuteCommand(cyberdaemon.Install)
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
	case cyberdaemon.NotInstalled:
		return InstallError{
			reason:                     "Daemon installation failed for unknown reason",
			daemonInstallFailedUnknown: true,
		}
	case cyberdaemon.Stopped:
		_, err := d.ExecuteCommand(cyberdaemon.Start)
		if err != nil {
			return InstallError{
				reason:            "Failed to start daemon after install - " + err.Error(),
				daemonStartFailed: true,
			}
		}
	}

	return nil
}

func Uninstall(d cyberdaemon.Daemon) error {
	status, statusErr := d.Status()
	if statusErr == nil && status == cyberdaemon.NotInstalled {
		return nil
	}

	if status == cyberdaemon.Running {
		_, err := d.ExecuteCommand(cyberdaemon.Stop)
		if err != nil {
			return UninstallError{
				reason:           "Failed to stop running daemon before uninstall",
				daemonStopFailed: true,
			}
		}
	}

	_, err := d.ExecuteCommand(cyberdaemon.Uninstall)
	if err != nil {
		return UninstallError{
			reason:                err.Error(),
			daemonUninstallFailed: true,
		}
	}

	return nil
}
