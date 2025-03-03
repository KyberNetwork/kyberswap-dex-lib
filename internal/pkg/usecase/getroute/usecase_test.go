package getroute

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	finderEngine "github.com/KyberNetwork/pathfinder-lib/pkg/finderengine"
	"github.com/KyberNetwork/pathfinder-lib/pkg/finderengine/finder/spfav2"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	routerpoolpkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/mocks/usecase/buildroute"
	"github.com/KyberNetwork/router-service/internal/pkg/mocks/usecase/getroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/poolfactory"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/safetyquote"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/mempool"
)

func TestGetRouteUseCase_Handle(t *testing.T) {
	t.Parallel()

	amountIn, _ := new(big.Int).SetString("1000000000000000", 10)

	testCases := []struct {
		name    string
		command dto.GetRoutesQuery
		err     error
	}{
		{
			name: "it should succeed when get route with common params",
			command: dto.GetRoutesQuery{
				TokenIn:    "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				TokenOut:   "0xdac17f958d2ee523a2206206994597c13d831ec7",
				AmountIn:   amountIn,
				SaveGas:    false,
				GasInclude: true,
				Index:      "nativeTvl",
			},
		},
		{
			name: "it should return poolSetFiltered when exclude sources",
			command: dto.GetRoutesQuery{
				TokenIn:         "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				TokenOut:        "0xdac17f958d2ee523a2206206994597c13d831ec7",
				AmountIn:        amountIn,
				SaveGas:         false,
				GasInclude:      true,
				ExcludedSources: []string{"uniswap"},
				Index:           "nativeTvl",
			},
			err: ErrPoolSetFiltered,
		},
		{
			name: "it should return poolSetFiltered when exclude pools",
			command: dto.GetRoutesQuery{
				TokenIn:       "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				TokenOut:      "0xdac17f958d2ee523a2206206994597c13d831ec7",
				AmountIn:      amountIn,
				SaveGas:       false,
				GasInclude:    true,
				ExcludedPools: mapset.NewThreadUnsafeSet("0x0d4a11d5eeaac28ec3f61d100daf4d40471f1852"),
				Index:         "nativeTvl",
			},
			err: ErrPoolSetFiltered,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := prepareUsecase(ctrl)
			result, err := uc.Handle(context.Background(), tc.command)

			assert.ErrorIs(t, err, tc.err)

			if tc.err == nil {
				assert.Equal(t, tc.command.TokenIn, result.RouteSummary.TokenIn)
				assert.Equal(t, tc.command.TokenOut, result.RouteSummary.TokenOut)
			}
		})
	}
}

func prepareUsecase(ctrl *gomock.Controller) *useCase {
	// Mock up tokens
	tokenWETH := &entity.Token{
		Name:     "Wrapped Ether",
		Symbol:   "WETH",
		Address:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
		Decimals: 18,
	}
	priceWETH := &routerEntity.OnchainPrice{
		USDPrice: routerEntity.Price{
			Buy:  big.NewFloat(1576.07366),
			Sell: big.NewFloat(1576.07366),
		},
	}
	tokenUSDT := &entity.Token{
		Name:     "Tether USD",
		Symbol:   "USDT",
		Address:  "0xdac17f958d2ee523a2206206994597c13d831ec7",
		Decimals: 6,
	}
	priceUSDT := &routerEntity.OnchainPrice{
		USDPrice: routerEntity.Price{
			Buy:  big.NewFloat(1),
			Sell: big.NewFloat(1),
		},
	}

	// Mock up pools
	entityPools := []*entity.Pool{
		{
			Address:  "0x0d4a11d5eeaac28ec3f61d100daf4d40471f1852",
			Exchange: "uniswap",
			Type:     "uniswap",
			Reserves: entity.PoolReserves{"21807913977161779085372", "34113654675815"},
			Tokens: []*entity.PoolToken{
				{Address: tokenWETH.Address, Weight: 50, Swappable: true},
				{Address: tokenUSDT.Address, Weight: 50, Swappable: true},
			},
		},
	}
	poolFactory := poolfactory.NewPoolFactory(poolfactory.Config{}, nil, nil, nil)
	pools := poolFactory.NewPools(context.Background(), entityPools, common.Hash{})

	// Mock IPoolRankRepository
	poolRankRepository := getroute.NewMockIPoolRankRepository(ctrl)
	poolRankRepository.EXPECT().
		FindBestPoolIDs(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(lo.Map(entityPools, func(item *entity.Pool, _ int) string { return item.Address }), nil).
		AnyTimes()

	// Mock ITokenRepository
	tokenRepository := buildroute.NewMockITokenRepository(ctrl)
	tokenRepository.EXPECT().
		FindByAddresses(gomock.Any(), gomock.Any()).
		Return([]*entity.Token{tokenWETH, tokenUSDT}, nil).
		AnyTimes()

	onchainpriceRepo := getroute.NewMockIOnchainPriceRepository(ctrl)
	onchainpriceRepo.EXPECT().FindByAddresses(gomock.Any(), gomock.Any()).
		Return(
			map[string]*routerEntity.OnchainPrice{
				tokenWETH.Address: priceWETH,
				tokenUSDT.Address: priceUSDT}, nil,
		).AnyTimes()

	// Mock IRouteCacheRepository
	routeCacheRepository := getroute.NewMockIRouteCacheRepository(ctrl)
	routeCacheRepository.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("some error")).
		AnyTimes()

	// Mock IGasRepository
	gasRepository := getroute.NewMockIGasRepository(ctrl)
	gasRepository.EXPECT().
		GetSuggestedGasPrice(gomock.Any()).
		Return(big.NewInt(7901274685), nil).
		AnyTimes()

	l1FeeEstimator := getroute.NewMockIL1FeeEstimator(ctrl)

	// Mock IPoolManager
	poolManager := getroute.NewMockIPoolManager(ctrl)
	poolManager.EXPECT().
		GetStateByPoolAddresses(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, addresses, dex []string, stateRoot common.Hash, extraData types.PoolManagerExtraData) (*types.FindRouteState, error) {
				addressesSet := mapset.NewThreadUnsafeSet(addresses...)
				dexesSet := mapset.NewThreadUnsafeSet(dex...)
				tokenToPoolAddress := make(map[string]*types.AddressList)

				var limits = make(map[string]map[string]*big.Int)
				limits[pooltypes.PoolTypes.KyberPMM] = make(map[string]*big.Int)
				limits[pooltypes.PoolTypes.Synthetix] = make(map[string]*big.Int)
				limits[pooltypes.PoolTypes.NativeV1] = make(map[string]*big.Int)
				limits[pooltypes.PoolTypes.LimitOrder] = make(map[string]*big.Int)
				filteredPools := make([]pool.IPoolSimulator, 0, len(pools))
				for _, pool := range pools {

					for _, tokenAddress := range pool.GetTokens() {
						if _, ok := tokenToPoolAddress[tokenAddress]; !ok {
							tokenToPoolAddress[tokenAddress] = mempool.AddressListPool.Get().(*types.AddressList)
						}
						tokenToPoolAddress[tokenAddress].AddAddress(ctx, pool.GetAddress())
					}
					if !addressesSet.Contains(pool.GetAddress()) {
						continue
					}
					if !dexesSet.Contains(pool.GetExchange()) {
						continue
					}
					filteredPools = append(filteredPools, pool)
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

				return &types.FindRouteState{
					Pools: lo.Associate(filteredPools, func(item pool.IPoolSimulator) (string, pool.IPoolSimulator) {
						return item.GetAddress(), item
					}),
					SwapLimit: poolFactory.NewSwapLimit(limits, types.PoolManagerExtraData{}),
				}, nil
			},
		).
		AnyTimes()
	poolManager.EXPECT().GetAEVMClient().Return(nil).AnyTimes()

	// Mock IBestPathRepository
	bestPathRepository := getroute.NewMockIBestPathRepository(ctrl)
	bestPathRepository.EXPECT().
		GetBestPaths(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	finderOptions := valueobject.FinderOptions{
		Type:                    valueobject.FinderTypes.SPFAv2,
		MaxHops:                 3,
		DistributionPercent:     5,
		MaxPathsInRoute:         20,
		MaxPathsToGenerate:      5,
		MaxPathsToReturn:        200,
		MinPartUSD:              500,
		MinThresholdAmountInUSD: 0,
		MaxThresholdAmountInUSD: 100000000,
	}

	calcAmountOutInstance := routerpoolpkg.NewCustomFuncs(map[string]bool{})

	routeFinder, _ := spfav2.NewSPFAv2Finder(
		finderOptions.MaxHops,
		finderOptions.MaxPathsToGenerate,
		finderOptions.MaxPathsToReturn,
		finderOptions.MaxPathsInRoute,
		finderOptions.DistributionPercent,
		finderOptions.MinPartUSD,
	)
	routeFinder.SetCustomFuncs(calcAmountOutInstance)

	routeFinalizer := findroute.NewSafetyQuotingRouteFinalizer(
		safetyquote.NewSafetyQuoteReduction(&valueobject.SafetyQuoteReductionConfig{}),
		calcAmountOutInstance,
	)

	finderEngine := finderEngine.NewPathFinderEngine(routeFinder, routeFinalizer)

	return NewUseCase(
		poolRankRepository,
		tokenRepository,
		onchainpriceRepo,
		routeCacheRepository,
		gasRepository,
		l1FeeEstimator,
		poolManager,
		finderEngine,
		Config{
			ChainID:          valueobject.ChainIDEthereum,
			GasTokenAddress:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			AvailableSources: lo.Map(entityPools, func(item *entity.Pool, _ int) string { return item.Exchange }),

			Aggregator: AggregatorConfig{
				WhitelistedTokenSet: map[string]bool{
					"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": true,
					"0xdefa4e8a7bcba345f687a2f1456f5edd9ce97202": true,
				},
				FinderOptions: finderOptions,
			},

			SafetyQuoteConfig: &valueobject.SafetyQuoteReductionConfig{},
		},
	)
}

func Test_useCase_getSources(t *testing.T) {
	type fields struct {
		config Config
	}
	type args struct {
		includedSources     []string
		excludedSources     []string
		onlyScalableSources bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []string
	}{
		{
			name: "default",
			fields: fields{
				config: Config{
					AvailableSources: []string{"a", "b", "c"},
				},
			},
			args: args{
				includedSources:     nil,
				excludedSources:     nil,
				onlyScalableSources: false,
			},
			want: []string{"a", "b", "c"},
		},
		{
			name: "default onlyScalableSources",
			fields: fields{
				config: Config{
					AvailableSources:  []string{"a", "b", "c"},
					UnscalableSources: []string{"a"},
				},
			},
			args: args{
				includedSources:     nil,
				excludedSources:     nil,
				onlyScalableSources: true,
			},
			want: []string{"b", "c"},
		},
		{
			name: "included",
			fields: fields{
				config: Config{
					AvailableSources: []string{"a", "b", "c"},
				},
			},
			args: args{
				includedSources:     []string{"a", "b"},
				excludedSources:     nil,
				onlyScalableSources: false,
			},
			want: []string{"a", "b"},
		},
		{
			name: "included but onlyScalableSources",
			fields: fields{
				config: Config{
					AvailableSources:  []string{"a", "b", "c"},
					UnscalableSources: []string{"a"},
				},
			},
			args: args{
				includedSources:     []string{"a", "b"},
				excludedSources:     nil,
				onlyScalableSources: true,
			},
			want: []string{"b"},
		},
		{
			name: "excluded",
			fields: fields{
				config: Config{
					AvailableSources: []string{"a", "b", "c"},
				},
			},
			args: args{
				includedSources:     nil,
				excludedSources:     []string{"a", "b"},
				onlyScalableSources: false,
			},
			want: []string{"c"},
		},
		{
			name: "excluded and onlyScalableSources",
			fields: fields{
				config: Config{
					AvailableSources:  []string{"a", "b", "c"},
					UnscalableSources: []string{"a"},
				},
			},
			args: args{
				includedSources:     nil,
				excludedSources:     []string{"b"},
				onlyScalableSources: true,
			},
			want: []string{"c"},
		},
		{
			name: "included and excluded",
			fields: fields{
				config: Config{
					AvailableSources: []string{"a", "b", "c"},
				},
			},
			args: args{
				includedSources:     []string{"a", "b"},
				excludedSources:     []string{"a"},
				onlyScalableSources: false,
			},
			want: []string{"b"},
		},
		{
			name: "included and excluded and onlyScalableSources",
			fields: fields{
				config: Config{
					AvailableSources:  []string{"a", "b", "c"},
					UnscalableSources: []string{"b"},
				},
			},
			args: args{
				includedSources:     []string{"a", "b", "c"},
				excludedSources:     []string{"a"},
				onlyScalableSources: true,
			},
			want: []string{"c"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &useCase{
				config: tt.fields.config,
			}
			got := u.getSources("", tt.args.includedSources, tt.args.excludedSources, tt.args.onlyScalableSources)
			assert.ElementsMatch(t, got, tt.want)
		})
	}
}
