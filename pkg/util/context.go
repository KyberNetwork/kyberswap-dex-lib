package util

import (
	"context"
	"time"
)

type contextKey string

const (
	KeyTimestamp contextKey = "timestamp"
)

func NewContextWithTimestamp(ctx context.Context) context.Context {
	return context.WithValue(ctx, KeyTimestamp, time.Now().Unix())
}
