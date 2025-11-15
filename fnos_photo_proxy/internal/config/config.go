package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	Port         string            `json:"port"`
	FnosBaseURL  string            `json:"fnos_base_url"`
	ImmichURL    string            `json:"immich_url"`
	ImmichAPIKey string            `json:"immich_api_key"`
	PathReplace  map[string]string `json:"path_replace"`
	SQLiteDBPath string            `json:"sqlite_db_path"`
	AutomateURL  string            `json:"automate_url"`
}

func LoadConfig() (*Config, error) {
	file, err := os.ReadFile("config.json")
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(file, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
