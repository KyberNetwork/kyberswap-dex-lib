package findroute

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/core"
	poolPkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
)

// IFinder is an interface of finding route algorithm
type IFinder interface {
	// Find performs finding route algorithm and return zero, one or multiple routes.
	// In case it returns multiple routes, the first route (index 0) is the best route
	Find(ctx context.Context, input Input, data FinderData) ([]*core.Route, error)
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
}

// FinderData contains all data for finding route.
type FinderData struct {
	// PoolByAddress mapping from pool address to IPool
	PoolByAddress map[string]poolPkg.IPool

	// TokenByAddress mapping from token address to token info (decimals, symbol, ...)
	TokenByAddress map[string]entity.Token

	// PriceUSDByAddress mapping from token address to price in USD
	PriceUSDByAddress map[string]float64
}
