package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Monitor MonitorConfig `yaml:"monitor"`
	Network NetworkConfig `yaml:"network"`
	Log     LogConfig     `yaml:"log"`
}

// ============ Monitor ===============
type MonitorModule struct {
	Enable   bool `yaml:"enable"`
	Interval int  `yaml:"interval"` // 秒
}

type MonitorConfig struct {
	Data   string        `yaml:"data"`
	Days   int           `yaml:"days"`
	CPU    MonitorModule `yaml:"cpu"`
	Memory MonitorModule `yaml:"memory"`
	Disk   MonitorModule `yaml:"disk"`
	Net    MonitorModule `yaml:"net"`
}

// ============= Log ================
type LogConfig struct {
	Debug      bool   `yaml:"debug"`
	Path       string `yaml:"path"`
	MaxSizeMB  int    `yaml:"max_size"`
	MaxAge     int    `yaml:"max_age"`
	MaxBackups int    `yaml:"max_backups"`
}

// ============= Network ================
type NetworkConfig struct {
	IgnoreOlder int      `yaml:"ignore_older"`
	Host        string   `yaml:"host"`
	Data        string   `yaml:"data"`
	Category    []string `yaml:"category"`
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
