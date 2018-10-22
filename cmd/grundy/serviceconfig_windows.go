package main

import (
	"github.com/kardianos/service"
)

func serviceConfig() (*service.Config, error) {
	return &service.Config{
		Name:        name,
		DisplayName: name,
		Description: description,
	}, nil
}
