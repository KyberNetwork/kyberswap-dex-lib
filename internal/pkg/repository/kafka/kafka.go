package kafka

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/IBM/sarama"
)

type Publisher struct {
	producer sarama.SyncProducer
}

func NewPublisher(config *Config) (*Publisher, error) {
	c := sarama.NewConfig()

	// Sarama Producer's MaxMessageBytes is currently compare to the size of
	// un-compress message, while it should compare to compressed size.
	// This causes error when publish some messages with large un-compress size,
	// while the compressed size is smaller than Broker's config.
	// However, this is just client-side safety check, remote Kafka cluster
	// will check it again with correct compressed size.
	// So, set MaxMessageBytes to MaxRequestSize for now.
	// Reference: https://github.com/IBM/sarama/issues/2142
	c.Producer.MaxMessageBytes = int(sarama.MaxRequestSize)
	c.Producer.Compression = sarama.CompressionLZ4
	c.Producer.RequiredAcks = sarama.WaitForAll
	c.Producer.Return.Successes = true
	c.Producer.Return.Errors = true

	c.Net.SASL.Enable = config.UseAuthentication
	c.Net.SASL.User = config.Username
	c.Net.SASL.Password = config.Password

	// Use SyncProducer since we want to ensure the message is published.
	producer, err := sarama.NewSyncProducer(config.Addresses, c)
	if err != nil {
		return nil, err
	}

	return &Publisher{producer: producer}, nil
}

func (k *Publisher) Publish(ctx context.Context, topic string, data []byte) error {
	message := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(data),
	}

	if _, _, err := k.producer.SendMessage(message); err != nil {
		return err
	}

	return nil
}

// ValidateTopicName returns error if the string is invalid as Kafka topic name.
// NOTE: Due to limitations in metric names, topics with a period ('.') or underscore
// ('_') could collide. To avoid issues it is best to use either, but not both.
func ValidateTopicName(topic string) error {
	expression := "^[a-zA-Z0-9\\._\\-]+$"
	matched, err := regexp.MatchString(expression, topic)
	if err != nil {
		return err
	}

	if !matched {
		return errors.New("invalid characters in topic name")
	}

	if strings.Contains(topic, "-") && strings.Contains(topic, ".") {
		return errors.New("collide characters in topic name")
	}

	return nil
}
