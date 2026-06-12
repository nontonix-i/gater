package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig      `yaml:"server"`
	Database DatabaseConfig    `yaml:"database"`
	Auth     AuthConfig        `yaml:"auth"`
	Upload   UploadConfig      `yaml:"upload"`
	Keepalive KeepaliveConfig `yaml:"keepalive"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

type DatabaseConfig struct {
	DSN string `yaml:"dsn"`
}

type AuthConfig struct {
	Enabled bool `yaml:"enabled"`
}

type UploadConfig struct {
	TempDir     string `yaml:"temp_dir"`
	MaxFileSize int64  `yaml:"max_file_size"`
}

type KeepaliveConfig struct {
	Enabled      bool `yaml:"enabled"`
	CheckEvery   int  `yaml:"check_every"`   // minutes between scheduler runs
	VisitOlder   int  `yaml:"visit_older"`   // days since last keepalive before re-visit
	RequestLimit int  `yaml:"request_limit"` // max URLs to visit per run
}

func Load(path string) (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{Port: 8080},
		Upload: UploadConfig{
			TempDir:     "/tmp/gater",
			MaxFileSize: 754974720,
		},
		Keepalive: KeepaliveConfig{
			Enabled:      true,
			CheckEvery:   1440, // 24h in minutes
			VisitOlder:   30,   // 30 days
			RequestLimit: 50,   // 50 URLs per run
		},
	}

	if data, err := os.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	if envDSN := os.Getenv("DATABASE_URL"); envDSN != "" {
		cfg.Database.DSN = envDSN
	}

	return cfg, nil
}
