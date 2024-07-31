package getcustomroute

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"

	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/getroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type aggregator struct {
	poolFactory            IPoolFactory
	tokenRepository        ITokenRepository
	priceRepository        IPriceRepository
	onchainpriceRepository IOnchainPriceRepository
	poolRepository         IPoolRepository

	finderEngine findroute.IFinderEngine
}

func NewCustomAggregator(
	poolFactory IPoolFactory,
	tokenRepository ITokenRepository,
	priceRepository IPriceRepository,
	onchainpriceRepository IOnchainPriceRepository,
	poolRepository IPoolRepository,
	finderEngine findroute.IFinderEngine,
) *aggregator {
	return &aggregator{
		poolFactory:            poolFactory,
		tokenRepository:        tokenRepository,
		priceRepository:        priceRepository,
		onchainpriceRepository: onchainpriceRepository,
		poolRepository:         poolRepository,
		finderEngine:           finderEngine,
	}
}

func (a *aggregator) Aggregate(ctx context.Context, params *types.AggregateParams, poolIds []string) (*valueobject.RouteSummary, error) {
	// Step 1: get pool set
	poolEntities, err := a.poolRepository.FindByAddresses(ctx, poolIds)
	if err != nil {
		return nil, err
	}

	if len(poolEntities) == 0 {
		return nil, getroute.ErrPoolSetEmpty
	}

	poolByAddress := make(map[string]poolpkg.IPoolSimulator, len(poolIds))
	poolInterfaces := a.poolFactory.NewPools(ctx, poolEntities, common.Hash{}) // Not use AEVM in custom route
	for i := range poolInterfaces {
		poolByAddress[poolInterfaces[i].GetAddress()] = poolInterfaces[i]
	}

	if len(poolByAddress) == 0 {
		return nil, getroute.ErrPoolSetFiltered
	}

	// Step 2: collect tokens and price data
	tokenAddresses := collectTokenAddresses(
		poolByAddress,
		params.TokenIn.Address,
		params.TokenOut.Address,
		params.GasToken.Address,
	)

	tokenByAddress, err := a.getTokenByAddress(ctx, tokenAddresses)
	if err != nil {
		return nil, err
	}

	var priceUSDByAddress map[string]float64

	// only get price from onchain-price-service if enabled
	var priceByAddress map[string]*routerEntity.OnchainPrice
	if a.onchainpriceRepository != nil {
		priceByAddress, err = a.onchainpriceRepository.FindByAddresses(ctx, tokenAddresses)
		if err != nil {
			return nil, err
		}
	} else {
		priceUSDByAddress, err = a.getPriceUSDByAddress(ctx, tokenAddresses)
		if err != nil {
			return nil, err
		}
	}

	var limits = make(map[string]map[string]*big.Int)
	limits[pooltypes.PoolTypes.KyberPMM] = make(map[string]*big.Int)
	limits[pooltypes.PoolTypes.Synthetix] = make(map[string]*big.Int)
	limits[pooltypes.PoolTypes.NativeV1] = make(map[string]*big.Int)
	limits[pooltypes.PoolTypes.LimitOrder] = make(map[string]*big.Int)
	for _, pool := range poolInterfaces {
		dexLimit, avail := limits[pool.GetType()]
		if !avail {
			continue
		}
		limitMap := pool.CalculateLimit()
		for k, v := range limitMap {
			if old, exist := dexLimit[k]; !exist || old.Cmp(v) < 0 {
				dexLimit[k] = v
			}
		}
	}

	// Step 3: finds best route
	return a.findBestRoute(ctx, params, tokenByAddress, priceUSDByAddress, priceByAddress, &types.FindRouteState{
		Pools:     poolByAddress,
		SwapLimit: a.poolFactory.NewSwapLimit(limits),
	})
}

func (a *aggregator) ApplyConfig(config getroute.Config) {}

// findBestRoute find the best route and summarize it
func (a *aggregator) findBestRoute(
	ctx context.Context,
	params *types.AggregateParams,
	tokenByAddress map[string]*entity.Token,
	priceUSDByAddress map[string]float64,
	priceByAddress map[string]*routerEntity.OnchainPrice,
	state *types.FindRouteState,
) (*valueobject.RouteSummary, error) {
	input := findroute.Input{
		TokenInAddress:   params.TokenIn.Address,
		TokenOutAddress:  params.TokenOut.Address,
		AmountIn:         params.AmountIn,
		GasPrice:         params.GasPrice,
		GasTokenPriceUSD: params.GasTokenPriceUSD,
		TokenInPriceUSD:  params.TokenInPriceUSD,
		SaveGas:          params.SaveGas,
		GasInclude:       params.GasInclude,
	}

	data := findroute.NewFinderData(ctx, tokenByAddress, priceUSDByAddress, priceByAddress, state)
	defer data.ReleaseResources()
	route, err := a.finderEngine.Find(ctx, input, data, params)
	if err != nil {
		return nil, errors.WithMessagef(getroute.ErrRouteNotFound, "find route failed: [%v]", err)
	}

	return route, nil
}

// getTokenByAddress receives a list of address and returns a map of address to entity.Token
func (a *aggregator) getTokenByAddress(ctx context.Context, tokenAddresses []string) (map[string]*entity.Token, error) {
	tokens, err := a.tokenRepository.FindByAddresses(ctx, tokenAddresses)
	if err != nil {
		return nil, err
	}

	tokenByAddress := make(map[string]*entity.Token, len(tokens))
	for _, token := range tokens {
		tokenByAddress[token.Address] = token
	}

	return tokenByAddress, nil
}

// getPriceUSDByAddress receives a list of address and returns a map of address to its preferred price in USD
func (a *aggregator) getPriceUSDByAddress(ctx context.Context, tokenAddresses []string) (map[string]float64, error) {
	prices, err := a.priceRepository.FindByAddresses(ctx, tokenAddresses)
	if err != nil {
		return nil, err
	}

	priceUSDByAddress := make(map[string]float64, len(prices))
	for _, price := range prices {
		priceUSD, _ := price.GetPreferredPrice()

		priceUSDByAddress[price.Address] = priceUSD
	}

	return priceUSDByAddress, nil
}
