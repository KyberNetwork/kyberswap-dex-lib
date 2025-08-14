package context

import (
	"context"

	"github.com/google/uuid"
)

func NewJobCtx(ctx context.Context) context.Context {
	return context.WithValue(ctx, jobIDContextKey, uuid.New().String())
}

func GetJobID(ctx context.Context) string {
	jobID, _ := ctx.Value(jobIDContextKey).(string)

	return jobID
}

func NewCtxFromValue(ctx context.Context, key CtxKey, value string) context.Context {
	return context.WithValue(ctx, key, value)
}
