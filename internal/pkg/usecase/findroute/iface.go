package findroute

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/huandu/go-clone"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

// IFinder is an interface of finding route algorithm
type IFinder interface {
	// Find performs finding route algorithm and return zero, one or multiple routes.
	// In case it returns multiple routes, the first route (index 0) is the best route
	Find(ctx context.Context, input Input, data FinderData) ([]*valueobject.Route, error)
}

// Input contains parameter specified by clients.
type Input struct {
	// TokenInAddress address of token to be swapped
	TokenInAddress string `json:"tokenInAddress"`

	// TokenOutAddress address of token to be received
	TokenOutAddress string `json:"tokenOutAddress"`

	// AmountIn amount of token to be swapped
	AmountIn *big.Int `json:"amountIn"`

	// GasPrice price of gas in wei
	GasPrice *big.Float

	// GasTokenPriceUSD price of gas token in USD
	GasTokenPriceUSD float64

	// SaveGas should we find routes with minimal gas consumed
	SaveGas bool

	// GasInclude should we consider gas price when finding optimal route
	GasInclude bool

	// IsPathGeneratorEnabled should we use pregen paths
	IsPathGeneratorEnabled bool

	// SourceHash hash sources dex input by fnv hashing func
	SourceHash uint64
}

// FinderData contains all data for finding route.
type FinderData struct {
	PoolBucket *valueobject.PoolBucket

	// TokenByAddress mapping from token address to token info (decimals, symbol, ...)
	TokenByAddress map[string]entity.Token

	// PriceUSDByAddress mapping from token address to price in USD
	PriceUSDByAddress map[string]float64

	//SwapLimits is the map of dextype - Swap limit
	SwapLimits map[string]poolpkg.SwapLimit
}

func NewFinderData(poolByAddress map[string]poolpkg.IPoolSimulator, swapLimits map[string]poolpkg.SwapLimit, tokenByAddress map[string]entity.Token, tokenPriceUSDByAddress map[string]float64) FinderData {
	return FinderData{
		PoolBucket:        valueobject.NewPoolBucket(poolByAddress),
		TokenByAddress:    tokenByAddress,
		PriceUSDByAddress: tokenPriceUSDByAddress,
		SwapLimits:        clone.Slowly(swapLimits).(map[string]poolpkg.SwapLimit),
	}
}
