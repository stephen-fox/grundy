package daemonw

func GetConfig(name string, description string) (Config, error) {
	return Config{
		Name:        name,
		Description: description,
	}, nil
}
