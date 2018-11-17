// +build !windows

package cyberdaemon

import (
	"os/user"
)

func GetConfig(name string, description string) (Config, error) {
	u, err := user.Current()
	if err != nil {
		return Config{}, err
	}

	return Config{
		Name:        name,
		Description: description,
		Username:    u.Name,
	}, nil
}
