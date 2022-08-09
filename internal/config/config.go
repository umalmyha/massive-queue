// Package config contains logic for building project configuration
package config

import (
	"time"

	"github.com/caarlos0/env/v6"
)

// KafkaConsumerConfig is configuration for kafka consumer
type KafkaConsumerConfig struct {
	PostgresConnString string        `env:"POSTGRES_URL"`
	BrokerAddr         string        `env:"KAFKA_BROKER_ADDR"`
	Topic              string        `env:"KAFKA_TOPIC"`
	Partition          int           `env:"KAFKA_TOPIC_PARTITION" envDefault:"0"`
	BatchMinBytes      int           `env:"KAFKA_BATCH_MIN_BYTES" envDefault:"1000"`
	BatchMaxBytes      int           `env:"KAFKA_BATCH_MAX_BYTES" envDefault:"5000000"`
	CommitInterval     time.Duration `env:"KAFKA_COMMIT_INTERVAL" envDefault:"1s"`
}

// KafkaProducerConfig is configuration for kafka producer
type KafkaProducerConfig struct {
	BrokerAddr string `env:"KAFKA_BROKER_ADDR"`
	Topic      string `env:"KAFKA_TOPIC"`
	Partition  int    `env:"KAFKA_TOPIC_PARTITION" envDefault:"0"`
	BatchSize  int    `env:"KAFKA_BATCH_SIZE" envDefault:"2000"`
}

// KafkaConsumer builds config for kafka consumer
func KafkaConsumer() (*KafkaConsumerConfig, error) {
	var cfg KafkaConsumerConfig
	opts := env.Options{RequiredIfNoDef: true}

	if err := env.Parse(&cfg, opts); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// KafkaProducer builds config for kafka producer
func KafkaProducer() (*KafkaProducerConfig, error) {
	var cfg KafkaProducerConfig
	opts := env.Options{RequiredIfNoDef: true}

	if err := env.Parse(&cfg, opts); err != nil {
		return nil, err
	}

	return &cfg, nil
}
