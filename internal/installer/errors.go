package installer

type InstallError struct {
	reason                     string
	noDaemonStatus             bool
	daemonInstallFailedUnknown bool
	daemonStartFailed          bool
}

func (o InstallError) Error() string {
	return o.reason
}

func (o InstallError) FailedToGetDaemonStatusAfterInstall() bool {
	return o.noDaemonStatus
}

func (o InstallError) DaemonInstallFailedForUnknownReason() bool {
	return o.daemonInstallFailedUnknown
}

type UninstallError struct {
	reason                string
	daemonStopFailed      bool
	daemonUninstallFailed bool
}

func (o UninstallError) Error() string {
	return o.reason
}

func (o UninstallError) FailedToStopRunningDaemon() bool {
	return o.daemonStopFailed
}

func (o UninstallError) DaemonUninstallFailed() bool {
	return o.daemonUninstallFailed
}
