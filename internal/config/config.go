package config

import (
	"log/slog"
	"os"
	"runner/internal/lib/sl"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

const (
	EnvLocal       = "local"
	EnvDevelopment = "dev"
	EnvProduction  = "prod"
)

type Config struct {
	Env      string   `yaml:"env" env-required:"true"`
	Postgres Postgres `yaml:"postgres" env-required:"true"`
}

type Postgres struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Name     string `yaml:"name"`
	Password string `yaml:"password"`
	ModeSSL  string `yaml:"sslmode"`
}

// MustLoad loads config to a new Config instance and return it
func MustLoad() *Config {
	_ = godotenv.Load()

	configPath := os.Getenv("CONFIG_PATH")

	if configPath == "" {
		slog.Error("missed CONFIG_PATH parameter")
		os.Exit(1)
	}

	var err error
	if _, err = os.Stat(configPath); os.IsNotExist(err) {
		slog.Error("config file does not exist", slog.String("path", configPath))
		os.Exit(1)
	}

	var config Config

	if err = cleanenv.ReadConfig(configPath, &config); err != nil {
		slog.Error("cannot read config", sl.Err(err))
		os.Exit(1)
	}

	return &config
}

func Empty() *Config {
	return &Config{}
}
