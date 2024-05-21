package pool

import (
	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
)

type IPoolClient interface {
	TrackFaultyPools(ctx context.Context, trackers []entity.FaultyPoolTracker) ([]string, error)
	GetFaultyPools(ctx context.Context, offset int64, count int64) ([]string, error)
}
