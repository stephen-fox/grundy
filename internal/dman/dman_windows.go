package dman

import (
	"errors"
	"time"
	"strconv"

	"github.com/stephen-fox/userutil"
	"golang.org/x/sys/windows/svc/mgr"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/registry"
)

type windowsManager struct{
	config Config
}

func (o *windowsManager) Config() Config {
	return o.config
}

func (o *windowsManager) DoManagementCommand(command string) (Status, error) {
	return doManagementCommand(o, command)
}

func (o *windowsManager) ReinstallAndStart() error {
	o.Remove()

	err := o.Install()
	if err != nil {
		return err
	}

	err = o.Start()
	if err != nil {
		return err
	}

	return nil
}

func (o *windowsManager) Install() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	c := mgr.Config{
		DisplayName: o.config.Name(),
		Description: o.config.Description(),
		StartType:   mgr.StartManual,
	}

	if o.config.IsAutoStart() {
		c.StartType = mgr.StartAutomatic
	}

	s, err := m.CreateService(o.config.Name(), o.config.ExePath(), c, o.config.Arguments()...)
	if err != nil {
		return err
	}
	defer s.Close()

	err = eventlog.InstallAsEventCreate(o.config.Name(), eventlog.Error|eventlog.Warning|eventlog.Info)
	if err != nil {
		s.Delete()
		return err
	}

	return nil
}

func (o *windowsManager) Remove() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(o.config.Name())
	if err != nil {
		return err
	}
	defer s.Close()

	err = s.Delete()
	if err != nil {
		return err
	}

	err = eventlog.Remove(o.config.Name())
	if err != nil {
		return err
	}

	return nil
}

func (o *windowsManager) Start() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(o.config.Name())
	if err != nil {
		return err
	}
	defer s.Close()

	err = s.Start()
	if err != nil {
		return err
	}

	return nil
}

func (o *windowsManager) Stop() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(o.config.Name())
	if err != nil {
		return err
	}
	defer s.Close()

	err = stopAndWait(s)
	if err != nil {
		return err
	}

	return nil
}

// stopAndWait by takama et al., copied from
// github.com/takama/daemon/daemon_windows.go
//
// commit: 7b0f9893e24934bbedef065a1768c33779951e7d
func stopAndWait(s *mgr.Service) error {
	// First stop the service. Then wait for the service to
	// actually stop before starting it.
	status, err := s.Control(svc.Stop)
	if err != nil {
		return err
	}

	timeDuration := time.Millisecond * 50

	timeout := time.After(getStopTimeout() + (timeDuration * 2))
	tick := time.NewTicker(timeDuration)
	defer tick.Stop()

	for status.State != svc.Stopped {
		select {
		case <-tick.C:
			status, err = s.Query()
			if err != nil {
				return err
			}
		case <-timeout:
			break
		}
	}
	return nil
}

// getStopTimeout by takama et al., copied from
// github.com/takama/daemon/daemon_windows.go
//
// commit: 7b0f9893e24934bbedef065a1768c33779951e7d
func getStopTimeout() time.Duration {
	// For default and paths see https://support.microsoft.com/en-us/kb/146092
	defaultTimeout := time.Millisecond * 20000
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control`, registry.READ)
	if err != nil {
		return defaultTimeout
	}
	sv, _, err := key.GetStringValue("WaitToKillServiceTimeout")
	if err != nil {
		return defaultTimeout
	}
	v, err := strconv.Atoi(sv)
	if err != nil {
		return defaultTimeout
	}
	return time.Millisecond * time.Duration(v)
}

func (o *windowsManager) Status() (Status, error) {
	m, err := mgr.Connect()
	if err != nil {
		return "", err
	}
	defer m.Disconnect()

	s, err := m.OpenService(o.config.Name())
	if err != nil {
		return "", err
	}
	defer s.Close()

	winStatus, err := s.Query()
	if err != nil {
		return "", err
	}

	switch winStatus.State {
	case svc.StopPending:
		return Stopping, nil
	case svc.Stopped:
		return Stopped, nil
	case svc.StartPending:
		return Starting, nil
	case svc.Running:
		return Running, nil
	case svc.ContinuePending:
		return Resuming, nil
	case svc.PausePending:
		return Pausing, nil
	case svc.Paused:
		return Paused, nil
	}

	return Unknown, nil
}

func NewManager(config Config) (Manager, error) {
	err := userutil.IsRoot()
	if err != nil {
		return &windowsManager{}, errors.New("Application must be run as Administrator to manage the daemon")
	}

	return &windowsManager{
		config: config,
	}, nil
}
