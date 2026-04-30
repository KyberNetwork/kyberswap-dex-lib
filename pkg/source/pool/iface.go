package pool

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
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
	Logs         []types.Log
	BlockHeaders map[uint64]entity.BlockHeader
}

type GetNewPoolStateWithOverridesParams struct {
	Logs      []types.Log
	Overrides map[common.Address]gethclient.OverrideAccount
}

type IPoolTrackerWithOverrides interface {
	GetNewPoolStateWithOverrides(ctx context.Context, p entity.Pool,
		params GetNewPoolStateWithOverridesParams) (entity.Pool, error)
}

type IPoolTracker interface {
	GetNewPoolState(ctx context.Context, p entity.Pool, params GetNewPoolStateParams) (entity.Pool, error)
}

type IPoolTrackerWithDependencies interface {
	GetDependencies(ctx context.Context, p entity.Pool) ([]string, bool, error)
	SetDependenciesStored(p *entity.Pool, isStored bool) error
}

type IPoolSimulator interface {
	// CalcAmountOut amountOut, fee, gas
	// the required params is TokenAmountIn and TokenOut.
	// SwapLimit is optional, individual dex logic will choose to ignore it if it is nil
	CalcAmountOut(params CalcAmountOutParams) (*CalcAmountOutResult, error)
	// CloneState clones IPoolSimulator to back up old balance state before UpdateBalance by a swap.
	// Only clones fields updated by UpdateBalance. Returns nil if unimplemented.
	CloneState() IPoolSimulator
	// UpdateBalance updates the pool state after a swap
	UpdateBalance(params UpdateBalanceParams)
	CanSwapTo(address string) []string
	CanSwapFrom(address string) []string
	GetTokens() []string
	GetReserves() []*big.Int
	GetAddress() string
	GetExchange() string
	GetType() string
	GetMetaInfo(tokenIn, tokenOut string) any
	GetTokenIndex(address string) int
	CalculateLimit() map[string]*big.Int
	// GetApprovalAddress returns the address that should be approved to spend tokenIn
	GetApprovalAddress(tokenIn, tokenOut string) string
}

type IPoolExactOutSimulator interface {
	// CalcAmountIn returns amountIn, fee, gas
	// the required params is TokenAmountOut and TokenIn.
	// SwapLimit is optional, individual dex logic will choose to ignore it if it is nil
	CalcAmountIn(param CalcAmountInParams) (*CalcAmountInResult, error)
}

type IMetaPoolSimulator interface {
	IPoolSimulator
	GetBasePools() []IPoolSimulator      // get base pools
	SetBasePool(basePool IPoolSimulator) // set base pool
}

type IPoolSupportNativeSwap interface {
	SwapReceiveNativeIn(tokenIn, tokenOut string, chainId valueobject.ChainID) bool
	SwapReturnNativeOut(tokenIn, tokenOut string, chainId valueobject.ChainID) bool
}

type (
	// ICustomFuncs provides customizable functions for calculating amount out and cloning pool states
	ICustomFuncs interface {
		ICustomCalcAmountOut
		ICustomClone
	}

	// ICustomCalcAmountOut can CalcAmountOut and customize this function
	ICustomCalcAmountOut interface {
		CalcAmountOut(ctx context.Context, pool IPoolSimulator, tokenAmountIn TokenAmount, tokenOut string,
			swapLimit SwapLimit) (*CalcAmountOutResult, error)
		SetCustomCalcAmountOutFunc(f CalcAmountOutFunc)
	}
	// ICustomClone can ClonePool and CloneSwapLimit and customize these functions
	ICustomClone interface {
		ClonePool(ctx context.Context, pool IPoolSimulator) IPoolSimulator
		SetCustomClonePoolFunc(f ClonePoolFunc)
		CloneSwapLimit(ctx context.Context, limit SwapLimit) SwapLimit
		SetCustomCloneSwapLimitFunc(f CloneSwapLimitFunc)
	}

	CalcAmountOutFunc func(ctx context.Context, pool IPoolSimulator, tokenAmountIn TokenAmount,
		tokenOut string, swapLimit SwapLimit) (*CalcAmountOutResult, error)
	ClonePoolFunc      func(ctx context.Context, pool IPoolSimulator) IPoolSimulator
	CloneSwapLimitFunc func(ctx context.Context, limit SwapLimit) SwapLimit
)

type IPoolSingleRFQ interface {
	RFQ(ctx context.Context, params RFQParams) (*RFQResult, error)
}

type IPoolRFQ interface {
	IPoolSingleRFQ
	BatchRFQ(ctx context.Context, paramsSlice []RFQParams) ([]*RFQResult, error)
	SupportBatch() bool
}

type IPoolDecoder interface {
	Decode(ctx context.Context, logs []types.Log) (addressLogs map[string][]types.Log, err error)
}

type ITBPoolTracker[T any] interface {
	FetchRPCData(ctx context.Context, pool *entity.Pool, blockNumber uint64) (T, error)
}

// ITicksBasedPoolTracker fetches ticks for pool from Swap, Mint and Burn events.
// GetNewPoolState (from IPoolTracker) applies log-based updates using params.Logs and params.BlockHeaders.
// BootstrapPoolState performs full RPC/subgraph refresh (e.g. when params have no logs).
type ITicksBasedPoolTracker interface {
	IPoolTracker
	BootstrapPoolState(ctx context.Context, p entity.Pool, params GetNewPoolStateParams) (entity.Pool, error)
	FetchPoolTicks(ctx context.Context, p entity.Pool) (entity.Pool, error)
}

type IPoolFactoryDecoder interface {
	DecodePoolCreated(event types.Log) (*entity.Pool, error)
	IsEventSupported(hash common.Hash) bool
}
