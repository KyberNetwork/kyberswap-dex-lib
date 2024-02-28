package consumer

import (
	"context"
)

// Handler is a function type for handling messages from stream consumer.
type Handler[T any] func(ctx context.Context, msg T) error

type Consumer[T any] interface {
	Consume(ctx context.Context, h Handler[T]) error
}
