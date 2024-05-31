package poolmanager

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/ethereum/go-ethereum/common"
)

//go:generate mockgen -destination ../../mocks/poolmanager/pool_factory.go -package poolmanager github.com/KyberNetwork/router-service/internal/pkg/usecase/poolmanager IPoolFactory
type IPoolFactory interface {
	NewPoolByAddress(ctx context.Context, pools []*entity.Pool, stateRoot common.Hash) map[string]poolpkg.IPoolSimulator
	NewPools(ctx context.Context, pools []*entity.Pool, stateRoot common.Hash) []poolpkg.IPoolSimulator
	NewSwapLimit(limits map[string]map[string]*big.Int) map[string]poolpkg.SwapLimit
}

//go:generate mockgen -destination ../../mocks/poolmanager/pool_repository.go -package poolmanager github.com/KyberNetwork/router-service/internal/pkg/usecase/poolmanager IPoolRepository
type IPoolRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Pool, error)
	GetFaultyPools(ctx context.Context) ([]string, error)
	GetPoolsInBlacklist(ctx context.Context) ([]string, error)
}

//go:generate mockgen -destination ../../mocks/poolmanager/pool_rank_repository.go -package poolmanager github.com/KyberNetwork/router-service/internal/pkg/usecase/poolmanager IPoolRankRepository
type IPoolRankRepository interface {
	FindGlobalBestPools(ctx context.Context, poolCount int64) []string
}
