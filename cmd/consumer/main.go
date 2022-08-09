package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/umalmyha/massive-queue/internal/broker"
	"github.com/umalmyha/massive-queue/internal/config"
	"github.com/umalmyha/massive-queue/internal/repository"
)

const pgConnectTimeout = 10 * time.Second

func main() {
	cfg, err := config.KafkaConsumer()
	if err != nil {
		logrus.Fatal(err)
	}

	pgPool, err := postgresql(cfg.PostgresConnString)
	if err != nil {
		logrus.Fatal(err)
	}
	defer pgPool.Close()

	reader := kafkaReader(cfg)

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt)

	messageRps := repository.NewPostgresMessageRepository(pgPool)

	ctx, cancel := context.WithCancel(context.Background())
	consumer := broker.NewKafkaMessageConsumer(reader, messageRps)

	go func() {
		logrus.Infof("starting consumption from kafka...")
		if err := consumer.Consume(ctx); err != nil {
			logrus.Errorf("error occurred while stopping kafka consumer - %v", err)
		}
	}()

	sig := <-stopCh
	logrus.Infof("signal %s has been sent, stopping kafka reader", sig.String())

	cancel()
	if err := reader.Close(); err != nil {
		logrus.Errorf("failed to stop reader gracefully - %v", err)
	}
}

func kafkaReader(cfg *config.KafkaConsumerConfig) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		MinBytes:       cfg.BatchMinBytes,
		MaxBytes:       cfg.BatchMaxBytes,
		Brokers:        []string{cfg.BrokerAddr},
		Topic:          cfg.Topic,
		Partition:      cfg.Partition,
		CommitInterval: cfg.CommitInterval,
	})
}

func postgresql(uri string) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), pgConnectTimeout)
	defer cancel()

	pool, err := pgxpool.Connect(ctx, uri)
	if err != nil {
		return nil, fmt.Errorf("failed to establish connection to db - %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("didn't get response from database after sending ping request - %w", err)
	}
	return pool, nil
}
