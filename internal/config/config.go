package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Registry RegistryConfig `yaml:"registry"`
	Monitor  MonitorConfig  `yaml:"monitor"`
	Log      LogConfig      `yaml:"log"`
}

type RegistryConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type MonitorConfig struct {
	CPU  bool `yaml:"cpu"`
	Disk bool `yaml:"disk"`
	Net  bool `yaml:"net"`
}

type LogConfig struct {
	Path       string `yaml:"path"`
	MaxSizeMB  int    `yaml:"max_size"`
	MaxAge     int    `yaml:"max_age"`
	MaxBackups int    `yaml:"max_backups"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
