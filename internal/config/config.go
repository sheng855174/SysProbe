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

// Registry
type RegistryConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// Monitor
type MonitorModule struct {
	Enable   bool `yaml:"enable"`
	Interval int  `yaml:"interval"` // 毫秒
}

type MonitorConfig struct {
	Data   string        `yaml:"data"`
	Days   int           `yaml:"days"`
	CPU    MonitorModule `yaml:"cpu"`
	Memory MonitorModule `yaml:"memory"`
	Disk   MonitorModule `yaml:"disk"`
	Net    MonitorModule `yaml:"net"`
}

// Log
type LogConfig struct {
	Debug      bool   `yaml:"debug"`
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
