package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	StoragePath   string        `yaml:"storage_path" env-required:"true"`
	GRPC          GRPCConfig    `yaml:"grpc"`
	Address       string        `yaml:"address"  env-default:"localhost:8080"`
	TokenTTL      time.Duration `yaml:"token_ttl" env-default:"1h"`
	SecretJWT     string        `yaml:"secret_jwt"`
	SecretStorage string        `yaml:"secret_storage"`
}

type GRPCConfig struct {
	Port    int           `yaml:"port"`
	Timeout time.Duration `yaml:"timeout"`
}

func InitConfig(path string) (*Config, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config Config
	err = yaml.Unmarshal(file, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}
