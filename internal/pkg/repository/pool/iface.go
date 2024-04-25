package pool

import "context"

type IPoolClient interface {
	TrackFaultyPools(ctx context.Context, poolAddresses []string) ([]string, error)
	GetFaultyPools(ctx context.Context, offset int64, count int64) ([]string, error)
}
