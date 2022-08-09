// Package broker contains logic for processing message brokers logic
package broker

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/umalmyha/massive-queue/internal/model"
	"github.com/umalmyha/massive-queue/internal/repository"
	"github.com/vmihailenco/msgpack/v5"
)

const (
	datasourceWriteTimeout = 5 * time.Second
	batchSize              = 2000
)

// MessageConsumer represents behavior of Message data consumer
type MessageConsumer interface {
	Consume(context.Context) error
}

type kafkaMessageConsumer struct {
	reader     *kafka.Reader
	messageRps repository.MessageRepository
}

// NewKafkaMessageConsumer builds kafka Message consumer
func NewKafkaMessageConsumer(r *kafka.Reader, messageRps repository.MessageRepository) MessageConsumer {
	return &kafkaMessageConsumer{reader: r, messageRps: messageRps}
}

// Consume starts blocking read of kafka topic
func (c *kafkaMessageConsumer) Consume(ctx context.Context) error {
	totalMessagesSaved := 0
	batch := c.messageRps.NewBatch()

Read:
	for {
		select {
		case <-ctx.Done():
			break Read
		default:
			if err := c.readMessage(ctx, batch); err != nil {
				if errors.Is(err, io.EOF) {
					break Read
				}
				logrus.Errorf("failed to process message - %v", err)
			}

			batchLen := batch.Len()
			if batchLen >= batchSize {
				logrus.Infof("batch size %d reached - saving to datasource", batchLen)
				if err := c.sendBatch(batch); err != nil {
					logrus.Errorf("failed to save last %d messages - %v", batchLen, err)
				}

				totalMessagesSaved += batchLen
				logrus.Infof("total messages saved %d", totalMessagesSaved)

				batch = c.messageRps.NewBatch()
			}
		}
	}

	// consumption stopped, but some messages can await save
	if batch.Len() > 0 {
		if err := c.sendBatch(batch); err != nil {
			logrus.Errorf("failed to save last %d messages - %v", batch.Len(), err)
		}
	}

	return nil
}

func (c *kafkaMessageConsumer) readMessage(ctx context.Context, batch repository.MessageBatch) error {
	m, err := c.reader.ReadMessage(ctx)
	if err != nil {
		return err
	}

	var msg model.Message
	if err := msgpack.Unmarshal(m.Value, &msg); err != nil {
		return err
	}

	batch.CreateMessage(&msg)
	return nil
}

func (c *kafkaMessageConsumer) sendBatch(b repository.MessageBatch) error {
	ctx, cancel := context.WithTimeout(context.Background(), datasourceWriteTimeout)
	defer cancel()

	return c.messageRps.SendBatch(ctx, b)
}
