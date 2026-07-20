package config

import (
	"log"
	"sync"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	IsDebug bool `env:"IS_DEBUG" env-default:"true"`
	Listen  struct {
		Port string `env:"PORT" env-default:"8083"` // User Service висит на 8083
	}
	PostgreSQL struct {
		URL string `env:"PG_URL_USER" env-default:"postgres://clen_user:clenshop@localhost:5433/clen_users?sslmode=disable"`
	}
	Redis struct {
		URL string `env:"REDIS_URL_USER" env-default:"redis://localhost:6379/1"`
	}
	JWT struct {
		Secret     string        `env:"JWT_SECRET" env-default:"asd@123#"`
		AccessTTL  time.Duration `env:"JWT_ACCESS_TTL" env-default:"15m"`
		RefreshTTL time.Duration `env:"JWT_REFRESH_TTL" env-default:"720h"`
	}
}

var (
	instance *Config
	once     sync.Once
)

func GetConfig() *Config {
	once.Do(func() {
		instance = &Config{}
		if err := cleanenv.ReadEnv(instance); err != nil {
			log.Fatalf("Ошибка чтения конфигурации: %s", err)
		}
	})
	return instance
}
