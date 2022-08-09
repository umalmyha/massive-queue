package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/umalmyha/massive-queue/internal/config"
	"github.com/umalmyha/massive-queue/internal/model"
	"github.com/umalmyha/massive-queue/internal/random"
	"github.com/vmihailenco/msgpack/v5"
)

const (
	totalNumberOfMessages = 1_000_000
	sendBatchSize         = 2000
	msgHeaderLength       = 50
	msgContentLength      = 100
)

func main() {
	cfg, err := config.KafkaProducer()
	if err != nil {
		logrus.Fatal(err)
	}

	writer := kafkaWriter(cfg)

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt)

	logrus.Infof("starting to generate %d messages", totalNumberOfMessages)
	kafkaMsgs := make([]kafka.Message, totalNumberOfMessages)
	for i := 0; i < totalNumberOfMessages; i++ {
		m := &model.Message{
			ID:      uuid.NewString(),
			Header:  random.String(msgHeaderLength),
			Content: random.String(msgContentLength),
		}

		enc, err := msgpack.Marshal(m)
		if err != nil {
			logrus.Fatal(err)
		}

		kafkaMsgs[i] = kafka.Message{
			Topic:     cfg.Topic,
			Partition: cfg.Partition,
			Value:     enc,
		}
	}
	logrus.Info("messages generation finished")

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(time.Second)
		startPos := 0
		endPos := startPos + sendBatchSize

		for endPos <= totalNumberOfMessages {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := writer.WriteMessages(ctx, kafkaMsgs[startPos:endPos]...); err != nil {
					logrus.Errorf("failed to deliver messages to kafka - %v", err)
				}
			}

			startPos += +sendBatchSize
			endPos = startPos + sendBatchSize

			logrus.Infof("%d messages were sent, %d left", startPos, totalNumberOfMessages-startPos)
		}

		stopCh <- os.Interrupt // finish if all messages are sent
	}()

	<-stopCh
	logrus.Info("stopping kafka writer...")

	cancel()
	if err := writer.Close(); err != nil {
		logrus.Errorf("failed to stop writer gracefully - %v", err)
	}
}

func kafkaWriter(cfg *config.KafkaProducerConfig) *kafka.Writer {
	return &kafka.Writer{
		Addr:      kafka.TCP(cfg.BrokerAddr),
		BatchSize: cfg.BatchSize,
	}
}
