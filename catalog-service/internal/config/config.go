package config

import (
	"log"
	"sync"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	IsDebug bool `env:"IS_DEBUG" env-default:"true"`
	Listen  struct {
		BindIP   string `env:"BIND_IP" env-default:"0.0.0.0"`
		HTTPPort string `env:"PORT" env-default:"8081"`
		GRPCPort string `env:"GRPC_PORT" env-default:"50051"`
	}
	PostgreSQL struct {
		URL string `env:"PG_URL" env-default:"postgres://clen_user:secret@localhost:5433/clen_catalog?sslmode=disable"`
	}
	Kafka struct {
		Brokers []string `env:"KAFKA_BROKERS" env-default:"localhost:9092"`
		Topic   string   `env:"KAFKA_TOPIC" env-default:"orders.created"`
		GroupID string   `env:"KAFKA_GROUP_ID" env-default:"catalog-service-group"`
	}

	MinIO struct {
		Endpoint        string `env:"MINIO_ENDPOINT" env-default:"localhost:9000"`
		AccessKeyID     string `env:"MINIO_ACCESS_KEY" env-default:"clen_admin"`
		SecretAccessKey string `env:"MINIO_SECRET_KEY" env-default:"clenshop"`
		BucketName      string `env:"MINIO_BUCKET" env-default:"clen-images"`
		UseSSL          bool   `env:"MINIO_USE_SSL" env-default:"false"`
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
