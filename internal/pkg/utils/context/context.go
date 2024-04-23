package context

import (
	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/utils/requestid"
	"github.com/google/uuid"
)

func NewJobCtx(ctx context.Context) context.Context {
	return context.WithValue(ctx, jobIDContextKey, uuid.New().String())
}

func GetJobID(ctx context.Context) string {
	jobID, _ := ctx.Value(jobIDContextKey).(string)

	return jobID
}

func NewBackgroundCtxWithReqId(ctx context.Context) context.Context {
	return requestid.SetRequestIDToContext(context.Background(), requestid.GetRequestIDFromCtx(ctx))
}
