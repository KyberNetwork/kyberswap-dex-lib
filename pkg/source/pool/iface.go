package pool

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/ethereum/go-ethereum/core/types"
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

type GetNewPoolStateParams struct {
	Logs []types.Log
}

type IPoolTracker interface {
	GetNewPoolState(ctx context.Context, p entity.Pool, params GetNewPoolStateParams) (entity.Pool, error)
}

type IPoolSimulator interface {
	// CalcAmountOut amountOut, fee, gas
	CalcAmountOut(
		tokenAmountIn TokenAmount,
		tokenOut string,
	) (*CalcAmountOutResult, error)
	UpdateBalance(params UpdateBalanceParams)
	CanSwapTo(address string) []string
	CanSwapFrom(address string) []string
	GetTokens() []string
	GetReserves() []*big.Int
	GetAddress() string
	GetExchange() string
	GetType() string
	GetMetaInfo(tokenIn string, tokenOut string) interface{}
	GetTokenIndex(address string) int
}

type RFQResult struct {
	NewAmountOut *big.Int
	Extra        any
}

type IPoolRFQ interface {
	RFQ(ctx context.Context, recipient string, params any) (RFQResult, error)
}
