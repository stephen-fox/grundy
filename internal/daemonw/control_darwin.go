package daemonw

import (
	"os"
	"os/signal"

	"github.com/stephen-fox/launchctlutil"
)

func ExecuteCommand(command Command, config Config) (string, error) {
	if command == Status {
		details, err := launchctlutil.CurrentStatus(config.Name)
		if err != nil {
			return "", err
		}

		switch details.Status {
		case launchctlutil.NotInstalled:
			return notInstalledStatus, nil
		case launchctlutil.NotRunning:
			return stoppedStatus, nil
		case launchctlutil.Running:
			return runningStatus, nil
		}

		return unknownStatus, nil
	}

	lconfig, err := toLaunchdConfig(config)
	if err != nil {
		return "", err
	}

	switch command {
	case Start:
		err = launchctlutil.Start(lconfig.GetLabel(), lconfig.GetKind())
		if err != nil {
			return "", err
		}

		return "", nil
	case Stop:
		err = launchctlutil.Stop(lconfig.GetLabel(), lconfig.GetKind())
		if err != nil {
			return "", err
		}

		return "", nil
	case Install:
		err = launchctlutil.Install(lconfig)
		if err != nil {
			return "", err
		}

		return "", nil
	case Uninstall:
		filePath, err := lconfig.GetFilePath()
		if err != nil {
			return "", err
		}

		err = launchctlutil.Remove(filePath, lconfig.GetKind())
		if err != nil {
			return "", err
		}

		return "", nil
	}

	return "", CommandError{
		isUnknown: true,
		command:   command,
	}
}

func BlockAndRun(logic ApplicationLogic, config Config) error {
	c := make(chan os.Signal)

	signal.Notify(c, os.Interrupt)
	defer signal.Stop(c)

	err := logic.Start()
	if err != nil {
		return err
	}

	<-c

	err = logic.Stop()
	if err != nil {
		return err
	}

	return nil
}

func toLaunchdConfig(config Config) (launchctlutil.Configuration, error) {
	exePath, err := os.Executable()
	if err != nil {
		return nil, err
	}

	lconfig, err := launchctlutil.NewConfigurationBuilder().
		SetKind(launchctlutil.UserAgent).
		SetLabel(config.Name).
		SetRunAtLoad(true).
		SetCommand(exePath).
		Build()
	if err != nil {
		return nil, err
	}

	return lconfig, nil
}
