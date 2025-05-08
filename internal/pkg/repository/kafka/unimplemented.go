package kafka

import "context"

type UnimplementedPublisher struct {
}

func NewUnimplementedPublisher() *UnimplementedPublisher {
	return &UnimplementedPublisher{}
}

func (k *UnimplementedPublisher) Publish(ctx context.Context, topic string, data []byte) error {
	return nil
}

func (k *UnimplementedPublisher) PublishMultiple(ctx context.Context, topic string, data [][]byte) error {
	return nil
}
