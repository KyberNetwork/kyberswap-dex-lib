package findroute

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/huandu/go-clone"

	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/mempool"
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
	TokenByAddress map[string]*entity.Token

	//TokenToPoolAddress store the adjacent list on our bfs traversal
	TokenToPoolAddress map[string]*types.AddressList

	// PriceUSDByAddress mapping from token address to price in USD
	PriceUSDByAddress map[string]float64

	// price in Native with decimal already factored in
	PriceNativeByAddress map[string]*routerEntity.OnchainPrice

	//SwapLimits is the map of dextype - Swap limit
	SwapLimits map[string]poolpkg.SwapLimit

	//originSwapLimits
	originSwapLimits map[string]poolpkg.SwapLimit
}

func NewFinderData(ctx context.Context, tokenByAddress map[string]*entity.Token, tokenPriceUSDByAddress map[string]float64, tokenPriceNativeByAddress map[string]*routerEntity.OnchainPrice, state *types.FindRouteState) FinderData {
	tokenToPoolAddress := make(map[string]*types.AddressList)
	for _, pool := range state.Pools {
		for _, tokenAddress := range pool.GetTokens() {
			if _, ok := tokenToPoolAddress[tokenAddress]; !ok {
				tokenToPoolAddress[tokenAddress] = mempool.AddressListPool.Get().(*types.AddressList)
			}
			tokenToPoolAddress[tokenAddress].AddAddress(ctx, pool.GetAddress())
		}
	}

	return FinderData{
		PoolBucket:         valueobject.NewPoolBucket(state.Pools),
		TokenByAddress:     tokenByAddress,
		TokenToPoolAddress: tokenToPoolAddress,
		PriceUSDByAddress:  tokenPriceUSDByAddress,
		SwapLimits:         clone.Slowly(state.SwapLimit).(map[string]poolpkg.SwapLimit),
		originSwapLimits:   clone.Slowly(state.SwapLimit).(map[string]poolpkg.SwapLimit),

		PriceNativeByAddress: tokenPriceNativeByAddress,
	}
}

func (f *FinderData) ReleaseResources() {
	for _, al := range f.TokenToPoolAddress {
		mempool.ReturnAddressList(al)
	}
}

// Refresh will deeply copy original swapLimit and clear poolBucket.
func (f *FinderData) Refresh() {
	f.PoolBucket.ClearChangedPools()
	f.SwapLimits = clone.Slowly(f.originSwapLimits).(map[string]poolpkg.SwapLimit)
}

func (f *FinderData) TokenNativeBuyPrice(address string) *big.Float {
	if price, ok := f.PriceNativeByAddress[address]; ok {
		return price.NativePriceRaw.Buy
	}
	return nil
}
