package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const configFileName = ".gatorconfig.json"

type Config struct {
	DbUrl           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

func ReadConfig() (Config, error) {
	var config Config
	configPath, err := getConfigFilePath()
	if err != nil {
		return config, err
	}
	configFile, err := os.ReadFile(configPath)
	if err != nil {
		return config, errors.New("Could not read gator config file")
	}

	err = json.Unmarshal(configFile, &config)
	if err != nil {
		return Config{}, errors.New("Could not decode config to json")
	}

	return config, nil
}

func (cfg *Config) SetUser(username string) error {
	cfg.CurrentUserName = username

	err := writeConfig(cfg)
	if err != nil {
		return errors.New("Error setting current user")
	}

	return nil
}

func writeConfig(cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return errors.New("Error with turning config into json data")
	}

	configFilePath, err := getConfigFilePath()
	if err != nil {
		return errors.New("Error with getting config path")
	}

	err = os.WriteFile(configFilePath, data, 0644)
	if err != nil {
		return errors.New("Error writing to config file")
	}
	return nil
}

func getConfigFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", errors.New("No home directory found")
	}

	configPath := filepath.Join(homeDir, configFileName)
	return configPath, nil
}
