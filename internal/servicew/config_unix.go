// +build !windows

package servicew

import (
	"os/user"

	"github.com/kardianos/service"
)

func Config(name string, description string) (*service.Config, error) {
	u, err := user.Current()
	if err != nil {
		return &service.Config{}, err
	}

	config := &service.Config{
		Name:        name,
		DisplayName: name,
		Description: description,
		UserName:    u.Name,
	}

	return config, nil
}
