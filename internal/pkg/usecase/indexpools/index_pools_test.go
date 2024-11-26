package indexpools

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	mocks "github.com/KyberNetwork/router-service/internal/pkg/mocks/usecase/indexpools"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/poolrank"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/business"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
)

func mockPoolTokensTestIndexPools() []*entity.PoolToken {
	token1 := &entity.PoolToken{
		Address:   "tokenaddress1",
		Name:      "tokenName1",
		Symbol:    "tokenSymbol1",
		Decimals:  6,
		Weight:    50,
		Swappable: true,
	}
	token2 := &entity.PoolToken{
		Address:   "tokenaddress2",
		Name:      "tokenName2",
		Symbol:    "tokenSymbol2",
		Decimals:  6,
		Weight:    50,
		Swappable: true,
	}
	token3 := &entity.PoolToken{
		Address:   "tokenaddress3",
		Name:      "tokenName3",
		Symbol:    "tokenSymbol3",
		Decimals:  6,
		Weight:    50,
		Swappable: false,
	}
	token4 := &entity.PoolToken{
		Address:   "tokenaddress4",
		Name:      "tokenName4",
		Symbol:    "tokenSymbol4",
		Decimals:  6,
		Weight:    50,
		Swappable: true,
	}
	token5 := &entity.PoolToken{
		Address:   "tokenaddress5",
		Name:      "tokenName5",
		Symbol:    "tokenSymbol5",
		Decimals:  6,
		Weight:    50,
		Swappable: true,
	}
	return []*entity.PoolToken{token1, token2, token3, token4, token5}
}

func mockOnchainPrices() map[string]*routerEntity.OnchainPrice {
	price1 := &routerEntity.OnchainPrice{
		USDPrice: routerEntity.Price{
			Buy:  big.NewFloat(0.5),
			Sell: big.NewFloat(1),
		},
		NativePriceRaw: routerEntity.Price{
			Buy:  big.NewFloat(1),
			Sell: big.NewFloat(1),
		},
	}
	price2 := &routerEntity.OnchainPrice{
		USDPrice: routerEntity.Price{
			Buy:  big.NewFloat(1),
			Sell: big.NewFloat(1),
		},
		NativePriceRaw: routerEntity.Price{
			Buy:  big.NewFloat(1),
			Sell: big.NewFloat(1),
		},
	}
	price3 := &routerEntity.OnchainPrice{
		USDPrice: routerEntity.Price{
			Buy:  big.NewFloat(1),
			Sell: big.NewFloat(1),
		},
		NativePriceRaw: routerEntity.Price{
			Buy:  big.NewFloat(1),
			Sell: big.NewFloat(1),
		},
	}
	price4 := &routerEntity.OnchainPrice{
		USDPrice: routerEntity.Price{
			Buy:  big.NewFloat(1),
			Sell: big.NewFloat(1),
		},
		NativePriceRaw: routerEntity.Price{
			Buy:  big.NewFloat(1),
			Sell: big.NewFloat(1),
		},
	}
	price5 := &routerEntity.OnchainPrice{
		USDPrice: routerEntity.Price{
			Buy:  big.NewFloat(1),
			Sell: big.NewFloat(1),
		},
		NativePriceRaw: routerEntity.Price{
			Buy:  big.NewFloat(1),
			Sell: big.NewFloat(1),
		},
	}
	return map[string]*routerEntity.OnchainPrice{
		"tokenaddress1": price1,
		"tokenaddress2": price2,
		"tokenaddress3": price3,
		"tokenaddress4": price4,
		"tokenaddress5": price5,
	}
}

func mockPoolsTestIndexPools() []*entity.Pool {
	poolTokens := mockPoolTokensTestIndexPools()
	return []*entity.Pool{
		{
			Address:     "pooladdress1",
			SwapFee:     100,
			Exchange:    "exchange1",
			Type:        "type1",
			Timestamp:   1658373335,
			Reserves:    []string{"1000000000000000000", "2000000000000000000", "0"},
			Tokens:      []*entity.PoolToken{poolTokens[0], poolTokens[1], poolTokens[2]},
			Extra:       "extra1",
			StaticExtra: "staticExtra1",
			TotalSupply: "30000",
		},
		{
			Address:     "pooladdress2",
			SwapFee:     200,
			Exchange:    "exchange2",
			Type:        pooltypes.PoolTypes.CurveMeta,
			Timestamp:   1658373335,
			Reserves:    []string{"2000000000000000000", "3000000000000000000"},
			Tokens:      []*entity.PoolToken{poolTokens[1], poolTokens[0]},
			Extra:       "extra2",
			StaticExtra: fmt.Sprintf(`{"underlyingTokens": ["%s", "%s", "%s"]}`, poolTokens[1].Address, poolTokens[3].Address, poolTokens[4].Address),
			TotalSupply: "5000000000000000000",
		},
		{
			Address:     "pooladdress3",
			ReserveUsd:  0,
			SwapFee:     300,
			Exchange:    "exchange3",
			Type:        pooltypes.PoolTypes.CurveAave,
			Timestamp:   1658373335,
			Reserves:    []string{"0", "0"},
			Tokens:      []*entity.PoolToken{poolTokens[0], poolTokens[1]},
			Extra:       "extra2",
			StaticExtra: fmt.Sprintf(`{"underlyingTokens": ["%s", "%s"]}`, poolTokens[3].Address, poolTokens[4].Address),
			TotalSupply: "30000",
		},
		{
			Address:     "pooladdress4",
			SwapFee:     300,
			Exchange:    "exchange3",
			Type:        pooltypes.PoolTypes.CurveAave,
			Timestamp:   1658373335,
			Reserves:    []string{"0", "0"},
			Tokens:      []*entity.PoolToken{poolTokens[0], poolTokens[1]},
			Extra:       "extra2",
			StaticExtra: fmt.Sprintf(`{"underlyingTokens": ["%s", "%s"]}`, poolTokens[3].Address, poolTokens[4].Address),
			TotalSupply: "30000",
		},
		{
			Address:      "pooladdress5",
			AmplifiedTvl: 30,
			SwapFee:      300,
			Exchange:     "pancake-v3",
			Type:         pooltypes.PoolTypes.PancakeV3,
			Timestamp:    1658373335,
			Reserves:     []string{"990096161416868", "1000000000000000000"},
			Tokens:       []*entity.PoolToken{poolTokens[0], poolTokens[1]},
			Extra:        `{"liquidity":1728825575337728263438306,"sqrtPriceX96":88400328422539208376907242502325,"tickSpacing":200,"tick":140353,"ticks":[{"index":-887200,"liquidityGross":1728825575337728263438306,"liquidityNet":1728825575337728263438306},{"index":887200,"liquidityGross":1728825575337728263438306,"liquidityNet":-1728825575337728263438306}]}`,
			StaticExtra:  `{"poolId":"0xb95ec1d6fb087ff65157ebd531f87a951dd85007"}`,
			TotalSupply:  "30000",
		},
	}
}

func mockPoolsNativeTVL(pools []*entity.Pool, nativePriceByToken map[string]*routerEntity.OnchainPrice) []float64 {
	ctx := context.TODO()
	nativeTVL0, _ := business.CalculatePoolTVL(ctx, pools[0], nativePriceByToken)
	nativeTVL1, _ := business.CalculatePoolTVL(ctx, pools[1], nativePriceByToken)
	nativeTVL2, _ := business.CalculatePoolTVL(ctx, pools[2], nativePriceByToken)
	nativeTVL3, _ := business.CalculatePoolTVL(ctx, pools[3], nativePriceByToken)
	nativeTVL4, _ := business.CalculatePoolTVL(ctx, pools[4], nativePriceByToken)

	return []float64{
		nativeTVL0,
		nativeTVL1,
		nativeTVL2,
		nativeTVL3,
		nativeTVL4,
	}

}

func mockPoolsAmplifiedNativeTVL(
	pools []*entity.Pool,
	nativePriceByToken map[string]*routerEntity.OnchainPrice,
	tvlNative []float64) []float64 {
	ctx := context.TODO()
	amplifiedTvlNative0, useTvl0, err0 := business.CalculatePoolAmplifiedTVL(ctx, pools[0], nativePriceByToken)
	if err0 == nil && useTvl0 {
		amplifiedTvlNative0 = tvlNative[0]
	}
	amplifiedTvlNative1, useTvl1, err1 := business.CalculatePoolAmplifiedTVL(ctx, pools[1], nativePriceByToken)
	if err1 == nil && useTvl1 {
		amplifiedTvlNative1 = tvlNative[1]
	}
	amplifiedTvlNative2, useTvl2, err2 := business.CalculatePoolAmplifiedTVL(ctx, pools[2], nativePriceByToken)
	if err2 == nil && useTvl2 {
		amplifiedTvlNative2 = tvlNative[2]
	}
	amplifiedTvlNative3, useTvl3, err3 := business.CalculatePoolAmplifiedTVL(ctx, pools[3], nativePriceByToken)
	if err3 == nil && useTvl3 {
		amplifiedTvlNative3 = tvlNative[3]
	}
	amplifiedTvlNative4, useTvl4, err4 := business.CalculatePoolAmplifiedTVL(ctx, pools[4], nativePriceByToken)
	if err4 == nil && useTvl4 {
		amplifiedTvlNative4 = tvlNative[3]
	}

	return []float64{
		amplifiedTvlNative0,
		amplifiedTvlNative1,
		amplifiedTvlNative2,
		amplifiedTvlNative3,
		amplifiedTvlNative4,
	}

}

func TestIndexPools_Handle(t *testing.T) {
	t.Parallel()

	type TestCase struct {
		name    string
		prepare func(ctrl *gomock.Controller) *IndexPoolsUseCase
		command dto.IndexPoolsCommand
		result  *dto.IndexPoolsResult
	}

	theError := errors.New("some error")

	mockTokens := mockPoolTokensTestIndexPools()
	mockPools := mockPoolsTestIndexPools()
	mockPrices := mockOnchainPrices()
	mockNativeTvls := mockPoolsNativeTVL(mockPools, mockPrices)
	mockAmplifiedNativeTvls := mockPoolsAmplifiedNativeTVL(mockPools, mockPrices, mockNativeTvls)

	testCases := []TestCase{
		{
			name: "it should return nil result when no pool failed to index",
			prepare: func(ctrl *gomock.Controller) *IndexPoolsUseCase {
				mockConfig := IndexPoolsConfig{
					WhitelistedTokenSet: map[string]bool{
						mockTokens[0].Address: true,
						mockTokens[2].Address: true,
						mockTokens[3].Address: true,
					},
					ChunkSize: 100,
				}

				mockPoolRepo := mocks.NewMockIPoolRepository(ctrl)
				mockPoolRepo.EXPECT().
					FindByAddresses(
						gomock.Any(),
						[]string{
							mockPools[0].Address,
							mockPools[1].Address,
							mockPools[2].Address,
							mockPools[3].Address,
							"pooladdress6",
						},
					).Return([]*entity.Pool{mockPools[0], mockPools[1], mockPools[2], mockPools[3]}, nil)

				mockPoolRankRepo := mocks.NewMockIPoolRankRepository(ctrl)
				mockPoolRankRepo.EXPECT().AddToSortedSet(
					gomock.Any(),
					mockTokens[0].Address,
					mockTokens[1].Address,
					true,
					false,
					poolrank.SortByTVLNative, mockPools[0].Address, mockNativeTvls[0], true,
				).Return(nil)
				mockPoolRankRepo.EXPECT().AddToSortedSet(
					gomock.Any(),
					mockTokens[0].Address,
					mockTokens[1].Address,
					true,
					false,
					poolrank.SortByAmplifiedTVLNative, mockPools[0].Address, mockAmplifiedNativeTvls[0], false,
				).Return(nil)
				mockPoolRankRepo.EXPECT().AddToSortedSet(
					gomock.Any(),
					mockTokens[1].Address,
					mockTokens[0].Address,
					false,
					true,
					poolrank.SortByTVLNative, mockPools[1].Address, mockNativeTvls[1], true,
				).Return(nil)
				mockPoolRankRepo.EXPECT().AddToSortedSet(
					gomock.Any(),
					mockTokens[1].Address,
					mockTokens[0].Address,
					false,
					true,
					poolrank.SortByAmplifiedTVLNative, mockPools[1].Address, mockAmplifiedNativeTvls[1], false,
				).Return(nil)
				// for curve meta
				mockPoolRankRepo.EXPECT().AddToSortedSet(
					gomock.Any(),
					mockTokens[1].Address,
					mockTokens[3].Address,
					false,
					true,
					poolrank.SortByTVLNative, mockPools[1].Address, mockNativeTvls[1], true,
				).Return(nil)
				mockPoolRankRepo.EXPECT().AddToSortedSet(
					gomock.Any(),
					mockTokens[1].Address,
					mockTokens[3].Address,
					false,
					true,
					poolrank.SortByAmplifiedTVLNative, mockPools[1].Address, mockAmplifiedNativeTvls[1], false,
				).Return(nil)
				mockPoolRankRepo.EXPECT().AddToSortedSet(
					gomock.Any(),
					mockTokens[1].Address,
					mockTokens[4].Address,
					false,
					false,
					poolrank.SortByTVLNative, mockPools[1].Address, mockNativeTvls[1], true,
				).Return(nil)
				mockPoolRankRepo.EXPECT().AddToSortedSet(
					gomock.Any(),
					mockTokens[1].Address,
					mockTokens[4].Address,
					false,
					false,
					poolrank.SortByAmplifiedTVLNative, mockPools[1].Address, mockAmplifiedNativeTvls[1], false,
				).Return(nil)

				mockPoolRankRepo.EXPECT().
					GetDirectIndexLength(gomock.Any(), poolrank.SortByTVLNative, gomock.Any(), gomock.Any()).
					Return(int64(0), nil).AnyTimes()

				onchainPriceRepo := mocks.NewMockIOnchainPriceRepository(ctrl)
				onchainPriceRepo.EXPECT().FindByAddresses(gomock.Any(), gomock.All()).Return(mockPrices, nil).AnyTimes()

				return NewIndexPoolsUseCase(mockPoolRepo, mockPoolRankRepo, onchainPriceRepo, mockConfig)
			},
			command: dto.IndexPoolsCommand{PoolAddresses: []string{
				mockPools[0].Address,
				mockPools[1].Address,
				mockPools[2].Address,
				mockPools[3].Address,
				"pooladdress6",
			}},
			result: dto.NewIndexPoolsResult(nil, 0),
		},
		{
			name: "it should return correct failed pool addresses when some pools were failed to index",
			prepare: func(ctrl *gomock.Controller) *IndexPoolsUseCase {
				mockConfig := IndexPoolsConfig{
					WhitelistedTokenSet: map[string]bool{
						mockTokens[0].Address: true,
						mockTokens[2].Address: true,
						mockTokens[3].Address: true,
					},
					ChunkSize: 100,
				}

				mockPoolRepo := mocks.NewMockIPoolRepository(ctrl)
				mockPoolRepo.EXPECT().
					FindByAddresses(
						gomock.Any(),
						[]string{
							mockPools[0].Address,
							mockPools[1].Address,
							mockPools[2].Address,
							mockPools[3].Address,
							"pooladdress6",
						},
					).Return([]*entity.Pool{mockPools[0], mockPools[1], mockPools[2], mockPools[3]}, nil)

				mockPoolRankRepo := mocks.NewMockIPoolRankRepository(ctrl)
				mockPoolRankRepo.EXPECT().AddToSortedSet(
					gomock.Any(),
					mockTokens[0].Address,
					mockTokens[1].Address,
					true,
					false,
					poolrank.SortByTVLNative, mockPools[0].Address, mockNativeTvls[0], true,
				).Return(nil)
				mockPoolRankRepo.EXPECT().AddToSortedSet(
					gomock.Any(),
					mockTokens[0].Address,
					mockTokens[1].Address,
					true,
					false,
					poolrank.SortByAmplifiedTVLNative, mockPools[0].Address, mockAmplifiedNativeTvls[0], false,
				).Return(theError)
				mockPoolRankRepo.EXPECT().AddToSortedSet(
					gomock.Any(),
					mockTokens[1].Address,
					mockTokens[0].Address,
					false,
					true,
					poolrank.SortByTVLNative, mockPools[1].Address, mockNativeTvls[1], true,
				).Return(nil)
				mockPoolRankRepo.EXPECT().AddToSortedSet(
					gomock.Any(),
					mockTokens[1].Address,
					mockTokens[0].Address,
					false,
					true,
					poolrank.SortByAmplifiedTVLNative, mockPools[1].Address, mockAmplifiedNativeTvls[1], false,
				).Return(nil)
				// for curve meta
				mockPoolRankRepo.EXPECT().AddToSortedSet(
					gomock.Any(),
					mockTokens[1].Address,
					mockTokens[3].Address,
					false,
					true,
					poolrank.SortByTVLNative, mockPools[1].Address, mockNativeTvls[1], true,
				).Return(nil)
				mockPoolRankRepo.EXPECT().AddToSortedSet(
					gomock.Any(),
					mockTokens[1].Address,
					mockTokens[3].Address,
					false,
					true,
					poolrank.SortByAmplifiedTVLNative, mockPools[1].Address, mockAmplifiedNativeTvls[1], false,
				).Return(nil)
				mockPoolRankRepo.EXPECT().AddToSortedSet(
					gomock.Any(),
					mockTokens[1].Address,
					mockTokens[4].Address,
					false,
					false,
					poolrank.SortByTVLNative, mockPools[1].Address, mockNativeTvls[1], true,
				).Return(nil)
				mockPoolRankRepo.EXPECT().AddToSortedSet(
					gomock.Any(),
					mockTokens[1].Address,
					mockTokens[4].Address,
					false,
					false,
					poolrank.SortByAmplifiedTVLNative, mockPools[1].Address, mockAmplifiedNativeTvls[1], false,
				).Return(nil)
				mockPoolRankRepo.EXPECT().
					GetDirectIndexLength(gomock.Any(), poolrank.SortByTVLNative, gomock.Any(), gomock.Any()).
					Return(int64(0), nil).AnyTimes()
				onchainPriceRepo := mocks.NewMockIOnchainPriceRepository(ctrl)
				onchainPriceRepo.EXPECT().FindByAddresses(gomock.Any(), gomock.All()).Return(mockPrices, nil).AnyTimes()

				return NewIndexPoolsUseCase(mockPoolRepo, mockPoolRankRepo, onchainPriceRepo, mockConfig)
			},
			command: dto.IndexPoolsCommand{PoolAddresses: []string{
				mockPools[0].Address,
				mockPools[1].Address,
				mockPools[2].Address,
				mockPools[3].Address,
				"pooladdress6",
			}},
			result: dto.NewIndexPoolsResult([]string{"pooladdress1"}, 0),
		},
		{
			name: "it should return correct failed pool addresses when repository returns error",
			prepare: func(ctrl *gomock.Controller) *IndexPoolsUseCase {
				mockConfig := IndexPoolsConfig{
					ChunkSize: 100,
				}

				mockPoolRepo := mocks.NewMockIPoolRepository(ctrl)
				mockPoolRepo.EXPECT().
					FindByAddresses(gomock.Any(), []string{"pooladdress1", "pooladdress2", "pooladdress3"}).
					Return(nil, theError)
				mockPoolRankRepo := mocks.NewMockIPoolRankRepository(ctrl)
				return NewIndexPoolsUseCase(mockPoolRepo, mockPoolRankRepo, nil, mockConfig)
			},
			command: dto.IndexPoolsCommand{PoolAddresses: []string{"pooladdress1", "pooladdress2", "pooladdress3"}},
			result:  dto.NewIndexPoolsResult([]string{"pooladdress1", "pooladdress2", "pooladdress3"}, 0),
		},
		{
			name: "it should index 0 native TVL pools if number of direct pools is too small",
			prepare: func(ctrl *gomock.Controller) *IndexPoolsUseCase {
				mockConfig := IndexPoolsConfig{
					ChunkSize:                   100,
					MaxDirectIndexLenForZeroTvl: 10,
				}

				poolTokens := mockPoolTokensTestIndexPools()
				mockPool := &entity.Pool{
					Address:      "pooladdress5",
					ReserveUsd:   0,
					AmplifiedTvl: 0,
					SwapFee:      300,
					Exchange:     "exchange5",
					Type:         "type5",
					Timestamp:    1658373335,
					Reserves:     []string{"10000", "10000"},
					Tokens:       []*entity.PoolToken{poolTokens[0], poolTokens[1]},
					Extra:        "extra5",
					TotalSupply:  "30000",
				}

				mockPoolRepo := mocks.NewMockIPoolRepository(ctrl)
				mockPoolRepo.EXPECT().
					FindByAddresses(gomock.Any(), []string{mockPool.Address}).
					Return([]*entity.Pool{mockPool}, nil)
				mockPoolRankRepo := mocks.NewMockIPoolRankRepository(ctrl)
				mockTvl, _ := business.CalculatePoolTVL(context.TODO(), mockPool, nil)
				mockPoolRankRepo.EXPECT().
					AddToSortedSet(
						gomock.Any(),
						mockPool.Tokens[0].Address,
						mockPool.Tokens[1].Address,
						false,
						false,
						poolrank.SortByTVLNative,
						mockPool.Address,
						mockTvl,
						true,
					).Return(nil)
				mockPoolRankRepo.EXPECT().
					GetDirectIndexLength(gomock.Any(), poolrank.SortByTVLNative, mockPool.Tokens[0].Address, mockPool.Tokens[1].Address).
					Return(int64(0), nil)
				onchainPriceRepo := mocks.NewMockIOnchainPriceRepository(ctrl)
				onchainPriceRepo.EXPECT().FindByAddresses(gomock.Any(), gomock.All()).Return(nil, nil).AnyTimes()

				return NewIndexPoolsUseCase(mockPoolRepo, mockPoolRankRepo, onchainPriceRepo, mockConfig)
			},
			command: dto.IndexPoolsCommand{PoolAddresses: []string{"pooladdress5"}},
			result:  dto.NewIndexPoolsResult(nil, 0),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			indexPools := tc.prepare(ctrl)

			result := indexPools.Handle(context.Background(), tc.command)

			assert.Equal(t, tc.result, result)
		})
	}
}

func TestIndexPools_RemovePoolFromIndexes(t *testing.T) {
	t.Parallel()

	type TestCase struct {
		name    string
		prepare func(ctrl *gomock.Controller, p *entity.Pool) *IndexPoolsUseCase
		pool    *entity.Pool
		result  error
	}

	theError := errors.New("some error")

	mockTokens := mockPoolTokensTestIndexPools()
	mockPools := mockPoolsTestIndexPools()

	testCases := []TestCase{
		{
			name: "it should return nil when indexes is removed correctly with ReserveTvl",
			pool: mockPools[1],
			prepare: func(ctrl *gomock.Controller, p *entity.Pool) *IndexPoolsUseCase {
				mockConfig := IndexPoolsConfig{
					WhitelistedTokenSet: map[string]bool{
						mockTokens[0].Address: true,
						mockTokens[2].Address: true,
						mockTokens[3].Address: true,
					},
					ChunkSize: 100,
				}

				mockPoolRankRepo := mocks.NewMockIPoolRankRepository(ctrl)
				mockPoolRankRepo.EXPECT().RemoveFromSortedSet(
					gomock.Any(),
					mockTokens[1].Address,
					mockTokens[0].Address,
					false,
					true,
					poolrank.SortByTVLNative, mockPools[1].Address, true,
				).Return(nil)
				mockPoolRankRepo.EXPECT().RemoveFromSortedSet(
					gomock.Any(),
					mockTokens[1].Address,
					mockTokens[0].Address,
					false,
					true,
					poolrank.SortByAmplifiedTVLNative, mockPools[1].Address, false,
				).Return(nil)
				// for curve meta
				mockPoolRankRepo.EXPECT().RemoveFromSortedSet(
					gomock.Any(),
					mockTokens[1].Address,
					mockTokens[3].Address,
					false,
					true,
					poolrank.SortByTVLNative, mockPools[1].Address, true,
				).Return(nil)
				mockPoolRankRepo.EXPECT().RemoveFromSortedSet(
					gomock.Any(),
					mockTokens[1].Address,
					mockTokens[3].Address,
					false,
					true,
					poolrank.SortByAmplifiedTVLNative, mockPools[1].Address, false,
				).Return(nil)
				mockPoolRankRepo.EXPECT().RemoveFromSortedSet(
					gomock.Any(),
					mockTokens[1].Address,
					mockTokens[4].Address,
					false,
					false,
					poolrank.SortByTVLNative, mockPools[1].Address, true,
				).Return(nil)
				mockPoolRankRepo.EXPECT().RemoveFromSortedSet(
					gomock.Any(),
					mockTokens[1].Address,
					mockTokens[4].Address,
					false,
					false,
					poolrank.SortByAmplifiedTVLNative, mockPools[1].Address, false,
				).Return(nil)

				return NewIndexPoolsUseCase(nil, mockPoolRankRepo, nil, mockConfig)
			},
			result: nil,
		},
		{
			name: "it should return nil when indexes is removed correctly with AmplifiedReserveTvl",
			pool: mockPools[0],
			prepare: func(ctrl *gomock.Controller, p *entity.Pool) *IndexPoolsUseCase {
				mockConfig := IndexPoolsConfig{
					WhitelistedTokenSet: map[string]bool{
						mockTokens[0].Address: true,
						mockTokens[1].Address: true,
						mockTokens[3].Address: true,
					},
					ChunkSize: 100,
				}

				mockPoolRankRepo := mocks.NewMockIPoolRankRepository(ctrl)
				mockPoolRankRepo.EXPECT().RemoveFromSortedSet(
					gomock.Any(),
					mockTokens[0].Address,
					mockTokens[1].Address,
					true,
					true,
					poolrank.SortByTVLNative, p.Address, true,
				).Return(nil)
				mockPoolRankRepo.EXPECT().RemoveFromSortedSet(
					gomock.Any(),
					mockTokens[0].Address,
					mockTokens[1].Address,
					true,
					true,
					poolrank.SortByAmplifiedTVLNative, p.Address, false,
				).Return(nil)

				return NewIndexPoolsUseCase(nil, mockPoolRankRepo, nil, mockConfig)
			},
			result: nil,
		},
		{
			name: "it should return nil when indexes is removed correctly with AmplifiedReserveTvl and CurveAeva",
			pool: &entity.Pool{
				Address:      "pooladdress3",
				ReserveUsd:   10000,
				AmplifiedTvl: 30,
				SwapFee:      300,
				Exchange:     "exchange3",
				Type:         pooltypes.PoolTypes.CurveAave,
				Timestamp:    1658373335,
				Reserves:     []string{"3000", "7000"},
				Tokens:       []*entity.PoolToken{mockTokens[0], mockTokens[1]},
				Extra:        "extra2",
				StaticExtra:  fmt.Sprintf(`{"underlyingTokens": ["%s", "%s"]}`, mockTokens[3].Address, mockTokens[4].Address),
				TotalSupply:  "30000",
			},
			prepare: func(ctrl *gomock.Controller, p *entity.Pool) *IndexPoolsUseCase {
				mockConfig := IndexPoolsConfig{
					WhitelistedTokenSet: map[string]bool{
						mockTokens[0].Address: true,
						mockTokens[1].Address: true,
						mockTokens[3].Address: true,
					},
					ChunkSize: 100,
				}

				mockPoolRankRepo := mocks.NewMockIPoolRankRepository(ctrl)
				mockPoolRankRepo.EXPECT().RemoveFromSortedSet(
					gomock.Any(),
					mockTokens[0].Address,
					mockTokens[1].Address,
					true,
					true,
					poolrank.SortByTVLNative, p.Address, true,
				).Return(nil)
				mockPoolRankRepo.EXPECT().RemoveFromSortedSet(
					gomock.Any(),
					mockTokens[0].Address,
					mockTokens[1].Address,
					true,
					true,
					poolrank.SortByAmplifiedTVLNative, p.Address, false,
				).Return(nil)
				mockPoolRankRepo.EXPECT().RemoveFromSortedSet(
					gomock.Any(),
					mockTokens[3].Address,
					mockTokens[4].Address,
					true,
					false,
					poolrank.SortByTVLNative, p.Address, true,
				).Return(nil)
				mockPoolRankRepo.EXPECT().RemoveFromSortedSet(
					gomock.Any(),
					mockTokens[3].Address,
					mockTokens[4].Address,
					true,
					false,
					poolrank.SortByAmplifiedTVLNative, p.Address, false,
				).Return(nil)

				return NewIndexPoolsUseCase(nil, mockPoolRankRepo, nil, mockConfig)
			},
			result: nil,
		},
		{
			name: "it should return correct failed pool addresses when repository returns error",
			pool: &entity.Pool{
				Address:      "pooladdress3",
				ReserveUsd:   10000,
				AmplifiedTvl: 30,
				SwapFee:      300,
				Exchange:     "exchange3",
				Type:         pooltypes.PoolTypes.CurveAave,
				Timestamp:    1658373335,
				Reserves:     []string{"3000", "7000"},
				Tokens:       []*entity.PoolToken{mockTokens[0], mockTokens[1]},
				Extra:        "extra2",
				StaticExtra:  fmt.Sprintf(`{"underlyingTokens": ["%s", "%s"]}`, mockTokens[3].Address, mockTokens[4].Address),
				TotalSupply:  "30000",
			},
			prepare: func(ctrl *gomock.Controller, p *entity.Pool) *IndexPoolsUseCase {
				mockConfig := IndexPoolsConfig{
					WhitelistedTokenSet: map[string]bool{
						mockTokens[0].Address: true,
						mockTokens[1].Address: true,
						mockTokens[3].Address: true,
					},
					ChunkSize: 100,
				}

				mockPoolRankRepo := mocks.NewMockIPoolRankRepository(ctrl)
				mockPoolRankRepo.EXPECT().RemoveFromSortedSet(
					gomock.Any(),
					mockTokens[0].Address,
					mockTokens[1].Address,
					true,
					true,
					poolrank.SortByTVLNative, p.Address, true,
				).Return(theError).Times(1)
				mockPoolRankRepo.EXPECT().RemoveFromSortedSet(
					gomock.Any(),
					mockTokens[0].Address,
					mockTokens[1].Address,
					true,
					true,
					poolrank.SortByAmplifiedTVLNative, p.Address, false,
				).Return(nil)
				mockPoolRankRepo.EXPECT().RemoveFromSortedSet(
					gomock.Any(),
					mockTokens[3].Address,
					mockTokens[4].Address,
					true,
					false,
					poolrank.SortByTVLNative, p.Address, true,
				).Return(nil)
				mockPoolRankRepo.EXPECT().RemoveFromSortedSet(
					gomock.Any(),
					mockTokens[3].Address,
					mockTokens[4].Address,
					true,
					false,
					poolrank.SortByAmplifiedTVLNative, p.Address, false,
				).Return(nil)

				return NewIndexPoolsUseCase(nil, mockPoolRankRepo, nil, mockConfig)
			},
			result: theError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			indexPools := tc.prepare(ctrl, tc.pool)

			result := indexPools.RemovePoolFromIndexes(context.Background(), tc.pool)

			assert.Equal(t, tc.result, result)
		})
	}
}
