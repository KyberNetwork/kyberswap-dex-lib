package trackexecutor

import (
	"context"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/ethereum/go-ethereum/common"
)

type IExecutorBalanceRepository interface {
	// HasToken returns true if `executor` has more than 1 wei of `token`.
	// Formally, HasToken returns true if token.balanceOf(executor) > 0.
	HasToken(ctx context.Context, executorAddress string, queries []string) ([]bool, error)

	// HasPoolApproval returns true if `executor` approves max for `pool` to use `token`.
	// Formally, HasPoolApproval returns true if token.allowance(executor, pool) > 0.
	HasPoolApproval(ctx context.Context, executorAddress string, queries []dto.PoolApprovalQuery) ([]bool, error)

	// AddToken saves the info that `executor` has `token` in its balance.
	AddToken(ctx context.Context, executorAddress string, data []string) error

	// RemoveToken removes the info that `executor` has `token` in its balance.
	RemoveToken(ctx context.Context, executorAddress string, data []string) error

	// ApprovePool saves the info that `executor` approves `pool` to spend `token` on its behalf.
	ApprovePool(ctx context.Context, executorAddress string, data []dto.PoolApprovalQuery) error

	// GetLatestProcessedBlockNumber returns the latest processed block number when tracking executor.
	GetLatestProcessedBlockNumber(ctx context.Context, executorAddress string) (uint64, error)

	// UpdateLatestProcessedBlockNumber saves the latest processed block number when tracking executor.
	UpdateLatestProcessedBlockNumber(ctx context.Context, executorAddress string, blockNumber uint64) error
}

type IPoolFactory interface {
	NewPools(ctx context.Context, pools []*entity.Pool, stateRoot common.Hash) []poolpkg.IPoolSimulator
	NewPoolByAddress(ctx context.Context, pools []*entity.Pool, stateRoot common.Hash) map[string]poolpkg.IPoolSimulator
}

type IPoolRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Pool, error)
}
