package config

import (
	"log"
	"sync"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	IsDebug bool `env:"IS_DEBUG" env-default:"true"`
	Kafka   struct {
		Brokers []string `env:"KAFKA_BROKERS" env-default:"localhost:9092"`
		Topic   string   `env:"KAFKA_TOPIC" env-default:"orders.created"`
		GroupID string   `env:"KAFKA_GROUP_ID" env-default:"notification-service-group"`
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
