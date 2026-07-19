package config

import (
	"log"
	"sync"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	IsDebug bool `env:"IS_DEBUG" env-default:"true"`
	Listen  struct {
		Port string `env:"PORT" env-default:"8080"` //Единый порт для фронтенда
	}
	JWT struct {
		Secret string `env:"JWT_SECRET" env-default:"asd@123#"`
	}
	Services struct {
		Catalog string `env:"CATALOG_URL" env-default:"http://localhost:8081"`
		Order   string `env:"ORDER_URL" env-default:"http://localhost:8082"`
		User    string `env:"USER_URL" env-default:"http://localhost:8083"`
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
