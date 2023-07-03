package pool

import (
	"context"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type IPoolsListUpdater interface {
	// GetNewPools returns list of new pools
	// @param ctx context.Context
	// @param metadataBytes []byte the arbitrary metadata that liquidity source needs to perform its fetching round
	// @return []entity.Pool list of new pools
	// @return []byte the new metadataBytes for the next round
	// @return error if there is any error
	GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error)
}

type IPoolTracker interface {
	GetNewPoolState(ctx context.Context, p entity.Pool) (entity.Pool, error)
}

type IPoolSimulator interface {
	// CalcAmountOut amountOut, fee, gas
	// DO NOT FUCKING MODIFY THE POOL whilst calculating amount out. Call UpdateBalance when you need to.
	CalcAmountOut(
		tokenAmountIn TokenAmount,
		tokenOut string,
	) (*CalcAmountOutResult, error)
	UpdateBalance(params UpdateBalanceParams)
	CanSwapTo(address string) []string
	CanSwapFrom(address string) []string
	GetTokens() []string
	GetAddress() string
	GetExchange() string
	GetType() string
	GetMetaInfo(tokenIn string, tokenOut string) interface{}
	GetTokenIndex(address string) int
}
