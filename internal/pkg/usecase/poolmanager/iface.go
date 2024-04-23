package poolmanager

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/ethereum/go-ethereum/common"
)

type IPoolFactory interface {
	NewPoolByAddress(ctx context.Context, pools []*entity.Pool, stateRoot common.Hash) map[string]poolpkg.IPoolSimulator
	NewPools(ctx context.Context, pools []*entity.Pool, stateRoot common.Hash) []poolpkg.IPoolSimulator
	NewSwapLimit(limits map[string]map[string]*big.Int) map[string]poolpkg.SwapLimit
}

type IPoolRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Pool, error)
	GetPoolsInBlacklist(ctx context.Context) ([]string, error)
	GetFaultyPools(ctx context.Context, startTime, offset, count int64) ([]string, error)
}

type IPoolRankRepository interface {
	FindGlobalBestPools(ctx context.Context, poolCount int64) []string
}
