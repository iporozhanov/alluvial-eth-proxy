package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
	"log"
	"strings"
	"sync"
	"time"
)

type Config struct {
	HTTPPort         string        `env:"PORT" env-default:"8080"`
	HTTPTimeoutLimit time.Duration `env:"TIMEOUT_LIMIT" env-default:"5s"`
	LogLevel         string        `env:"LOG_LEVEL" env-default:"info"`
	ClientUrlString  string        `env:"CLIENT_URLS" env-required:"true"`
}

var instance *Config
var once sync.Once

func Instance() *Config {
	once.Do(func() {
		instance = &Config{}
		if err := godotenv.Load(); err != nil {
			log.Println("no .env file found")
		}

		if err := cleanenv.ReadEnv(instance); err != nil {
			log.Fatal(err)
		}

	})

	return instance
}

func (cfg *Config) ClientUrls() []string {
	return strings.Split(cfg.ClientUrlString, ",")
}
