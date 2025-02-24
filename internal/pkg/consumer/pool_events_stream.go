package consumer

import (
	"context"
	"os"
	"time"

	"github.com/KyberNetwork/pool-service/pkg/message"
	"github.com/dranikpg/gtrs"
	"github.com/goccy/go-json"
	"github.com/redis/go-redis/v9"
	"github.com/samber/lo"

	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

const PoolEvents = "pool_events_stream_consumer"

type StreamMessage map[string]*message.EventMessage

func (m *StreamMessage) FromMap(data map[string]any) error {
	*m = make(StreamMessage)
	for k, v := range data {
		var msg message.EventMessage
		if err := json.Unmarshal([]byte(v.(string)), &msg); err != nil {
			return err
		}
		(*m)[k] = &msg
	}
	return nil
}

type Config struct {
	Prefix        string        `mapstructure:"prefix"`
	Stream        string        `mapstructure:"stream"`
	Group         string        `mapstructure:"group"`
	Key           string        `mapstructure:"key"`
	Block         time.Duration `mapstructure:"block"`
	Count         int64         `mapstructure:"count"`
	BufferSize    uint          `mapstructure:"bufferSize"`
	AckBufferSize uint          `mapstructure:"ackBufferSize"`
}

type PoolEventsStreamConsumer struct {
	rdb redis.Cmdable
	cfg *Config
}

func NewPoolEventsStreamConsumer(rdb redis.Cmdable, cfg *Config) *PoolEventsStreamConsumer {
	return &PoolEventsStreamConsumer{
		rdb: &IgnoreNilRedis{Cmdable: rdb},
		cfg: cfg,
	}
}

func (c *PoolEventsStreamConsumer) Consume(ctx context.Context, handler Handler[*message.EventMessage]) error {
	stream := utils.Join(c.cfg.Prefix, c.cfg.Stream)
	name, _ := lo.TryOr1(os.Hostname, c.cfg.Group)
	consumerGroup := gtrs.NewGroupConsumer[StreamMessage](ctx, c.rdb, c.cfg.Group, name, stream, "0",
		gtrs.GroupConsumerConfig{
			StreamConsumerConfig: gtrs.StreamConsumerConfig{
				Block:      c.cfg.Block,
				Count:      c.cfg.Count,
				BufferSize: c.cfg.BufferSize,
			},
			AckBufferSize: c.cfg.AckBufferSize,
		})
	defer func() {
		_ = consumerGroup.Close()
	}()

	for msg := range consumerGroup.Chan() {
		if err := msg.Err; err != nil {
			logger.WithFields(ctx,
				logger.Fields{
					"job.name": PoolEvents,
					"error":    err,
					"msg.id":   msg.ID,
				},
			).Error("consume failed")
			return err
		}

		if err := handler(ctx, msg.Data[c.cfg.Key]); err != nil {
			logger.WithFields(ctx,
				logger.Fields{
					"job.name": PoolEvents,
					"error":    err,
					"msg.id":   msg.ID,
				},
			).Error("handler failed")
		}

		consumerGroup.Ack(msg)
	}

	return nil
}
