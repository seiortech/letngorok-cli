package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	Token string `json:"token"`
}

func GetConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(homeDir, ".ngorok")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(configDir, "config.json"), nil
}

func SaveToken(token string) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	config := Config{Token: token}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

func LoadToken() (string, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // No config file yet, not an error
		}
		return "", err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return "", err
	}

	return config.Token, nil
}
