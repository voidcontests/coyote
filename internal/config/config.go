package config

import (
	"log/slog"
	"os"
	"time"

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
	HTTP     HTTP     `yaml:"http"`
	Postgres Postgres `yaml:"postgres" env-required:"true"`
	Redis    Redis    `yaml:"redis" env-required:"true"`
	Runner   Runner   `yaml:"runner" env-required:"true"`
}

type HTTP struct {
	Port        string        `yaml:"port" env-default:"7197"`
	Timeout     time.Duration `yaml:"timeout" env-default:"3s"`
	IdleTimeout time.Duration `yaml:"idle_timeout" env-default:"30s"`
}

type Postgres struct {
	Host     string `yaml:"host" env-required:"true"`
	Port     string `yaml:"port" env-required:"true"`
	User     string `yaml:"user" env-required:"true"`
	Name     string `yaml:"name" env-required:"true"`
	Password string `yaml:"password" env-required:"true"`
	ModeSSL  string `yaml:"sslmode" env-required:"true"`
}

type Redis struct {
	Addr     string `yaml:"addr" env-required:"true"`
	Password string `yaml:"password" env-required:"true"`
	DB       int    `yaml:"db"`
}

type Runner struct {
	DockerImage string `yaml:"docker_image" env-required:"true"`
	Channel     string `yaml:"channel" env-required:"true"`
}

func MustLoad() *Config {
	_ = godotenv.Load()

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		slog.Error("CONFIG_PATH environment variable is not set")
		os.Exit(1)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		slog.Error("config file does not exist", slog.String("path", configPath))
		os.Exit(1)
	}

	var config Config
	if err := cleanenv.ReadConfig(configPath, &config); err != nil {
		slog.Error("cannot read config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	return &config
}
