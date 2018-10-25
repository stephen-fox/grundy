package servicew

import (
	"github.com/kardianos/service"
)

func Config(name string, description string) (*service.Config, error) {
	return &service.Config{
		Name:        name,
		DisplayName: name,
		Description: description,
	}, nil
}
