package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const configFileName = ".gatorconfig.json"

type Config struct {
	DbUrl string `json:"db_url"`
}

func ReadConfig() (Config, error) {
	var config Config
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return config, errors.New("No home directory found")
	}

	configPath := filepath.Join(homeDir, configFileName)
	configFile, err := os.ReadFile(configPath)
	if err != nil {
		return config, errors.New("Could not read gator config file")
	}
	fmt.Print(string(configFile))

	return config, nil
}

func (cfg *Config) setUser() {
	return
}
