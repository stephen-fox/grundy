package daemonw

import (
	"github.com/kardianos/service"
)

type appLogicWrapper struct {
	logic ApplicationLogic
}

func (o *appLogicWrapper) Start(s service.Service) error {
	return o.logic.Start()
}

func (o *appLogicWrapper) Stop(s service.Service) error {
	return o.logic.Stop()
}

type dummyService struct {}

func (o *dummyService) Start(s service.Service) error {
	return nil
}

func (o *dummyService) Stop(s service.Service) error {
	return nil
}

func ExecuteCommand(command Command, config Config) (string, error) {
	if command == GetStatus {
		status, err := CurrentStatus(config)
		if err != nil {
			return "", err
		}

		return status.printableStatus(), nil
	}

	s, err := service.New(&dummyService{}, toServiceConfig(config))
	if err != nil {
		return "", err
	}

	err = service.Control(s, command.string())
	if err != nil {
		return "", err
	}

	return executedCommandMessage(command), nil
}

func CurrentStatus(config Config) (Status, error) {
	s, err := service.New(&dummyService{}, toServiceConfig(config))
	if err != nil {
		return Unknown, err
	}

	status, err := s.Status()
	if err != nil {
		return Unknown, err
	}

	switch status {
	case service.StatusRunning:
		return Running, nil
	case service.StatusStopped:
		return Stopped, nil
	}

	return Unknown, nil
}

func BlockAndRun(logic ApplicationLogic, config Config) error {
	s, err := service.New(toServiceInterface(logic), toServiceConfig(config))
	if err != nil {
		return err
	}

	err = s.Run()
	if err != nil {
		return err
	}

	return nil
}

func toServiceInterface(logic ApplicationLogic) service.Interface {
	return &appLogicWrapper{
		logic: logic,
	}
}

func toServiceConfig(config Config) *service.Config {
	return &service.Config{
		Name:        config.Name,
		DisplayName: config.Name,
		Description: config.Description,
		UserName:    config.Username,
	}
}
