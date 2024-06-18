package aevm

import (
	"context"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type IPoolsPublisher interface {
	PublishedPoolIDs(storageID string) map[string]struct{}
	PublishedPools(storageID string) map[string]poolpkg.IPoolSimulator
	Publish(ctx context.Context, pools map[string]poolpkg.IPoolSimulator) (string, error)
}
