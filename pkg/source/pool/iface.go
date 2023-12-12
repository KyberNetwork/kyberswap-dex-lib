package pool

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"

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

type GetNewPoolStateParams struct {
	Logs []types.Log
}

type IPoolTracker interface {
	GetNewPoolState(ctx context.Context, p entity.Pool, params GetNewPoolStateParams) (entity.Pool, error)
}

type IPoolSimulator interface {
	// CalcAmountOut amountOut, fee, gas
	// the required params is TokenAmountIn and TokenOut.
	// SwapLimit is optional, individual dex logic will chose to ignore it if it is nill
	CalcAmountOut(params CalcAmountOutParams) (*CalcAmountOutResult, error)
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
	CalculateLimit() map[string]*big.Int
}

type RFQResult struct {
	NewAmountOut *big.Int
	Extra        any
}

type IPoolReverseSimulator interface {
	// CalcAmountIn calculate the `amountIn` of `tokenIn` needed to get `tokenAmountOut`
	// caller might need to run `CalcAmountOut` again to determine if the returned `amountIn` is good enough
	CalcAmountIn(
		tokenAmountOut TokenAmount,
		tokenIn string,
	) (*big.Int, error)
}

type IPoolRFQ interface {
	RFQ(ctx context.Context, recipient string, params any) (RFQResult, error)
}
