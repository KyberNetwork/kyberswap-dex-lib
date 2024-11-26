package pool

import (
	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
)

//go:generate go run go.uber.org/mock/mockgen -destination ../../mocks/repository/pool/pool_client.go -package pool github.com/KyberNetwork/router-service/internal/pkg/repository/pool IPoolClient

type IPoolClient interface {
	TrackFaultyPools(ctx context.Context, trackers []entity.FaultyPoolTracker) ([]string, error)
	GetFaultyPools(ctx context.Context, offset int64, count int64) ([]string, error)
}
