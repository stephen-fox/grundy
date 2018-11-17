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

type windowsDaemon struct {
	config *service.Config
}

func (o *windowsDaemon) ExecuteCommand(command Command) (string, error) {
	if command == GetStatus {
		status, err := o.Status()
		if err != nil {
			return "", err
		}

		return status.printableStatus(), nil
	}

	s, err := service.New(&dummyService{}, o.config)
	if err != nil {
		return "", err
	}

	err = service.Control(s, command.string())
	if err != nil {
		return "", err
	}

	return executedCommandMessage(command), nil
}

func (o *windowsDaemon) Status() (Status, error) {
	s, err := service.New(&dummyService{}, o.config)
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

func (o *windowsDaemon) BlockAndRun(logic ApplicationLogic) error {
	s, err := service.New(toServiceInterface(logic), o.config)
	if err != nil {
		return err
	}

	err = s.Run()
	if err != nil {
		return err
	}

	return nil
}

func NewDaemon(config Config) (Daemon, error) {
	sconfig := &service.Config{
		Name:        config.Name,
		DisplayName: config.Name,
		Description: config.Description,
		UserName:    config.Username,
	}

	return &windowsDaemon{
		config: sconfig,
	}, nil
}

func toServiceInterface(logic ApplicationLogic) service.Interface {
	return &appLogicWrapper{
		logic: logic,
	}
}
