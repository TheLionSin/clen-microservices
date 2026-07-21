package config

import (
	"log"
	"sync"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	IsDebug bool `env:"IS_DEBUG" env-default:"true"`
	Listen  struct {
		Port string `env:"PORT" env-default:"8082"`
	}
	Redis struct {
		Address  string `env:"REDIS_URL" env-default:"localhost:6379"`
		Password string `env:"REDIS_PASSWORD" env-default:""`
		DB       int    `env:"REDIS_DB" env-default:"0"`
	}
	PostgreSQL struct {
		URL string `env:"PG_URL" env-default:"postgres://clen_user:secret@localhost:5433/clen_orders?sslmode=disable"`
	}
	Clients struct {
		CatalogGRPC string `env:"CATALOG_GRPC_URL" env-default:"localhost:50051"`
	}
	Kafka struct {
		Brokers []string `env:"KAFKA_BROKERS" env-default:"localhost:9092"`
		Topic   string   `env:"KAFKA_TOPIC" env-default:"orders.created"`
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
