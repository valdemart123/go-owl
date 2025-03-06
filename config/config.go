// Package config provides the configuration for framework.
package config

import (
	"encoding/json"
	"log"
	"os"
)

// Config struct represents the structure of owl.config
type Config struct {
	Browser struct {
		Type string `json:"type"`
	} `json:"browser"`
}

// LoadConfig reads and parses the JSON config file
func LoadConfig() Config {
	file, err := os.ReadFile("owl.config")
	if err != nil {
		log.Fatal("Failed to open config file:", err)
	}

	var conf Config
	if err := json.Unmarshal(file, &conf); err != nil {
		log.Fatal("Failed to parse config file:", err)
	}

	return conf
}

// LoadBrowserType retrieves the browser type from the config
func LoadBrowserType() string {
	return LoadConfig().Browser.Type
}