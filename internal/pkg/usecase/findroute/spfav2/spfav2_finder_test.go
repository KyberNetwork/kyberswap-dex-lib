package spfav2

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"testing"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uni "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/uniswap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testPool struct {
	name   string
	in     string
	inRev  int
	out    string
	outRev int
}

type testSwap struct {
	poolName  string
	amountIn  uint64
	amountOut uint64
}
type testPaths []testSwap

var (
	tokenByAddress = map[string]*entity.Token{
		"a":   {Address: "a"},
		"b":   {Address: "b"},
		"c":   {Address: "c"},
		"d":   {Address: "d"},
		"e":   {Address: "e"},
		"f":   {Address: "f"},
		"g":   {Address: "g"},
		"gas": {Address: "gas"},
	}

	pools = []testPool{
		{"pool-ab-1", "a", 10, "b", 10},
		{"pool-ab-2", "a", 20, "b", 20},
		{"pool-ac-1", "a", 10, "c", 10},
		{"pool-bc-1", "b", 10, "c", 10},
		{"pool-ad-1", "a", 10, "d", 10},
		{"pool-cd-1", "c", 15, "d", 15},
		{"pool-de-1", "d", 10, "e", 10},
		{"pool-ef-1", "e", 10, "f", 10},
		{"pool-fe-1", "f", 10, "g", 10},
	}

	/*
	   a <--[pool-ab-1]--> b
	   a <--[pool-ab-2]--> b
	                       b <--[pool-bc-1]--> c
	   a <--[pool-ac-1]----------------------> c
	                                           c <--[pool-cd-1]--> d
	   a <--[pool-ad-1]------------------------------------------> d
	                                                               d <--[pool-de-1]--> e
	                                                                                   e <--[pool-ef-1]--> f
	                                                                                                       f <--[pool-fg-1]--> g
	*/

	testCases = []struct {
		name          string
		tokenIn       string
		tokenOut      string
		amountIn      *big.Int
		saveGas       bool
		expectedPaths []testPaths
	}{
		// single hop
		{"a->b saveGas should use pool-ab-2", "a", "b", big.NewInt(1000), true,
			[]testPaths{{{"pool-ab-2", 1000, 19}}}}, // ab2 yield more than ab1
		{"a->b NOT saveGas but small amount should still use pool-ab-2", "a", "b", big.NewInt(900), false,
			[]testPaths{{{"pool-ab-2", 900, 19}}}},
		{"a->b NOT saveGas should use both pool-ab-", "a", "b", big.NewInt(1000), false,
			[]testPaths{{{"pool-ab-2", 500, 19}}, {{"pool-ab-1", 500, 9}}}},

		// multi hops
		{"a->d saveGas should use pool-ad-1", "a", "d", big.NewInt(1000), true,
			[]testPaths{{{"pool-ad-1", 1000, 9}}}},
		{"a->d NOT saveGas but small amount should still use pool-ad-1", "a", "d", big.NewInt(900), false,
			[]testPaths{{{"pool-ad-1", 900, 9}}}},
		{"a->d NOT saveGas should use many pools", "a", "d", big.NewInt(1000), false,
			[]testPaths{
				{{"pool-ad-1", 500, 9}},
				{{"pool-ac-1", 500, 9}, {"pool-cd-1", 9, 4}}, // ac1-cd1 yield more than ab2-bc1-cd1
			}},
		{"a->c NOT saveGas should use many pools and go through ab2", "a", "c", big.NewInt(1000), false,
			[]testPaths{
				{{"pool-ac-1", 500, 9}},
				{{"pool-ab-2", 500, 19}, {"pool-bc-1", 19, 6}}, // ab2 yield more than ab1
			}},
		{"a->d but cd1 is used twice so should be updated correspondingly", "a", "d", big.NewInt(1900), false,
			[]testPaths{
				{{"pool-ad-1", 760, 9}},
				{{"pool-ac-1", 570, 9}, {"pool-cd-1", 9, 4}},
				{{"pool-ab-2", 570, 19}, {"pool-bc-1", 19, 6}, {"pool-cd-1", 6, 1}}, // cd1 has been used above, so has lower yield here
			}},

		// this is expected behavior for spfav2: there will be 4 "best paths", 3 of them use pool-cd-1
		// when the 4th path get processed, pool-cd-1's reserveOut has already been decreased to 1, so cannot swap anymore
		// so the bestMultiPathRoute is nil and bestSinglePathRoute will be used
		// (this behavior is not ideal, and might be changed in the future)
		{"a->d large amount should fail multipath and fallback to singlepath", "a", "d", big.NewInt(2000), false,
			[]testPaths{{{"pool-ad-1", 2000, 9}}}},

		// no route
		{"a->g should not success: exceeding maxHop", "a", "g", big.NewInt(1000), false, nil},
	}

	maxHop                  uint32  = 3
	maxPathToGenerate       uint32  = 5
	maxPathToReturn         uint32  = 5
	distributionPercent     uint32  = 5
	maxPathsInRoute         uint32  = 20
	minPartUSD              float64 = 500
	minThresholdAmountInUSD float64 = 0
	maxThresholdAmountInUSD uint32  = 100000000
)

func TestFindRoute(t *testing.T) {

	poolEntities := lo.Map(pools, func(p testPool, _ int) entity.Pool {
		return entity.Pool{
			Address:  p.name,
			Tokens:   entity.PoolTokens{&entity.PoolToken{Address: p.in}, &entity.PoolToken{Address: p.out}},
			Reserves: entity.PoolReserves{strconv.Itoa(p.inRev), strconv.Itoa(p.outRev)},
		}
	})

	tokenAddressList := lo.MapToSlice(tokenByAddress, func(adr string, _ *entity.Token) string { return adr })
	priceUSDByAddress := lo.SliceToMap(tokenAddressList, func(adr string) (string, float64) { return adr, 1 })
	poolByAddress := lo.SliceToMap(poolEntities, func(poolEntity entity.Pool) (string, poolpkg.IPoolSimulator) {
		pool, _ := uni.NewPoolSimulator(poolEntity)
		return pool.GetAddress(), pool
	})

	finder := NewSPFAv2Finder(
		maxHop,
		nil,
		distributionPercent,
		maxPathsInRoute,
		maxPathToGenerate,
		maxPathToReturn,
		minPartUSD,
		minThresholdAmountInUSD,
		float64(maxThresholdAmountInUSD),
		func(sourceHash uint64, tokenIn, tokenOut string) []*entity.MinimalPath { return nil },
	)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params := &types.AggregateParams{
				TokenIn:          *tokenByAddress[tc.tokenIn],
				TokenOut:         *tokenByAddress[tc.tokenOut],
				GasToken:         *tokenByAddress["gas"],
				TokenInPriceUSD:  priceUSDByAddress[tc.tokenIn],
				TokenOutPriceUSD: priceUSDByAddress[tc.tokenOut],
				GasTokenPriceUSD: priceUSDByAddress["gas"],
				AmountIn:         tc.amountIn,
				Sources:          []string{},
				SaveGas:          tc.saveGas,
				GasInclude:       true,
				GasPrice:         big.NewFloat(1),
				ExtraFee:         valueobject.ZeroExtraFee,
			}

			input := findroute.Input{
				TokenInAddress:   params.TokenIn.Address,
				TokenOutAddress:  params.TokenOut.Address,
				AmountIn:         params.AmountIn,
				GasPrice:         params.GasPrice,
				GasTokenPriceUSD: params.GasTokenPriceUSD,
				SaveGas:          params.SaveGas,
				GasInclude:       params.GasInclude,
			}

			data := findroute.NewFinderData(context.Background(), tokenByAddress, priceUSDByAddress, &types.FindRouteState{
				Pools:     poolByAddress,
				SwapLimit: make(map[string]poolpkg.SwapLimit),
			})

			allRoutes, err := finder.Find(context.TODO(), input, data)

			if tc.expectedPaths == nil {
				// expect no path found
				if err == nil && len(allRoutes) > 0 {
					routesStr, _ := json.MarshalIndent(allRoutes, "", " ")
					fmt.Println("unexpected route", routesStr)
					t.FailNow()
				}
				return
			}

			require.Nil(t, err, "expected to found some routes")
			require.Equal(t, 1, len(allRoutes))
			routes := allRoutes[0]

			// first check number of possible paths
			require.Equal(t, len(tc.expectedPaths), len(routes.Paths))

			// then check each path
			lo.ForEach(lo.Zip2(tc.expectedPaths, routes.Paths), func(tp lo.Tuple2[testPaths, *valueobject.Path], _ int) {
				expectedPath := tp.A
				actualPath := tp.B.PoolAddresses
				// should have the expected number of pool along the path
				require.Equal(t, len(expectedPath), len(actualPath))
				lo.ForEach(lo.Zip2(expectedPath, actualPath), func(tp lo.Tuple2[testSwap, string], _ int) {
					expectedPool := tp.A
					actualPool := tp.B
					assert.Equal(t, expectedPool.poolName, actualPool)
				})

				// assert.Equal(t, expectedPool.amountIn, actualPool.Input.Amount.Uint64())
				// assert.Equal(t, expectedPool.amountOut, actualPool.Output.Amount.Uint64())
			})
		})
	}

}
