package hillclimb

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"testing"

	"github.com/KyberNetwork/blockchain-toolkit/float"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	uni "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/uniswap"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute/spfav2"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type testPool struct {
	name   string
	in     string
	inRev  int
	out    string
	outRev int
}

type testPaths struct {
	amountIn  uint64
	amountOut uint64
	pools     []string
}

type testcase struct {
	name          string
	tokenIn       string
	tokenOut      string
	amountIn      *big.Int
	saveGas       bool
	expectedPaths []testPaths
}

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
		{"pool-ac-1-threshold-571", "a", 10, "c", 10}, // special pool: if amountIn > 570 then output will be doubled (see below)
		{"pool-bc-1", "b", 10, "c", 10},
		{"pool-ad-1", "a", 10, "d", 10},
		{"pool-cd-1", "c", 15, "d", 15},

		{"pool-af-1", "a", 100, "f", 100},
		{"pool-af-2-threshold-2501", "a", 100, "f", 100},
		{"pool-af-3-threshold-2001", "a", 100, "f", 100},
		{"pool-af-4-threshold-1501", "a", 100, "f", 100},
		{"pool-af-5", "a", 100, "f", 100},
	}

	/*
	   a <--[pool-ab-1]--> b
	   a <--[pool-ab-2]--> b
	                       b <--[pool-bc-1]--> c
	   a <--[pool-ac-1]----------------------> c
	                                           c <--[pool-cd-1]--> d
	   a <--[pool-ad-1]------------------------------------------> d
	*/

	testCases = []testcase{
		// single path should be kept the same
		{"a->b saveGas should use pool-ab-2", "a", "b", big.NewInt(1000), true,
			[]testPaths{{1000, 19, []string{"pool-ab-2"}}}}, // ab2 yield more than ab1
		{"a->b NOT saveGas but small amount should still use pool-ab-2", "a", "b", big.NewInt(900), false,
			[]testPaths{{900, 19, []string{"pool-ab-2"}}}},
		{"a->b NOT saveGas should use both pool-ab-", "a", "b", big.NewInt(1000), false,
			[]testPaths{{500, 19, []string{"pool-ab-2"}}, {500, 9, []string{"pool-ab-1"}}}},
		{"a->d saveGas should use pool-ad-1", "a", "d", big.NewInt(1000), true,
			[]testPaths{{1000, 9, []string{"pool-ad-1"}}}},
		{"a->d NOT saveGas but small amount should still use pool-ad-1", "a", "d", big.NewInt(900), false,
			[]testPaths{{900, 9, []string{"pool-ad-1"}}}},

		// multi paths but cannot be optimized anymore
		// (in current hillclimbing implementation we won't increase last path input)
		{"a->d NOT saveGas", "a", "d", big.NewInt(1000), false,
			[]testPaths{
				{500, 9, []string{"pool-ad-1"}},
				{500, 5, []string{"pool-ac-1-threshold-571", "pool-cd-1"}}, // ac1-cd1 yield more than ab2-bc1-cd1
			}},

		// optimized with higher yield
		{"a->d but cd1 is used twice so should be updated correspondingly", "a", "d", big.NewInt(1900), false,
			[]testPaths{
				{760, 9, []string{"pool-ad-1"}},
				{589, 8, []string{"pool-ac-1-threshold-571", "pool-cd-1"}}, // spfav2 will allocate 570 to this path (yield 5), then hillclimb will increase it up (higher than threshold 571 below)
				{551, 1, []string{"pool-ab-2", "pool-bc-1", "pool-cd-1"}},  // spfav2 will allocate 570 to this path (yield 2), then hillclimb will sacrifice it for the higher path above
			}},

		// optimized with higher yield
		{"a->f", "a", "f", big.NewInt(10000), false,
			[]testPaths{
				{2500, 96, []string{"pool-af-1"}},                 // spfav2: (2500, 95)
				{2600, 192, []string{"pool-af-2-threshold-2501"}}, // spfav2: (2500, 95)
				{2100, 190, []string{"pool-af-3-threshold-2001"}}, // spfav2: (2000, 94)
				{1400, 93, []string{"pool-af-4-threshold-1501"}},  // spfav2: (1500, 93)
				{1400, 93, []string{"pool-af-5"}},                 // spfav2: (1500, 93)
			}},
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
	poolByAddress := lo.SliceToMap(poolEntities, func(poolEntity entity.Pool) (string, poolpkg.IPoolSimulator) {
		pool, _ := uni.NewPoolSimulator(poolEntity)
		if strings.Contains(poolEntity.Address, "-threshold-") {
			thresholdStr := strings.Split(poolEntity.Address, "-threshold-")[1]
			threshold, _ := strconv.Atoi(thresholdStr)
			return pool.GetAddress(), &ThresholdSim{pool, int64(threshold)}
		}
		return pool.GetAddress(), pool
	})

	baseFinder := spfav2.NewSPFAv2Finder(
		maxHop,
		nil,
		distributionPercent,
		maxPathsInRoute,
		maxPathToGenerate,
		maxPathToReturn,
		minPartUSD,
		minThresholdAmountInUSD,
		float64(maxThresholdAmountInUSD),
		map[string]bool{},
	)
	finder := NewHillClimbingFinder(1, 2, 500, baseFinder)

	for _, tc := range testCases {
		f := func(t *testing.T, tc testcase, priceUSDByAddress map[string]float64, priceInNative map[string]*big.Float) {
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
				GasPrice:         big.NewFloat(1000),
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

			priceByAddress := lo.MapValues(priceInNative, func(v *big.Float, _ string) *routerEntity.OnchainPrice {
				priceDecimals := new(big.Float).Quo(v, float.TenPow(18))
				return &routerEntity.OnchainPrice{
					NativePriceRaw: routerEntity.Price{Buy: v, Sell: v},
					NativePrice:    routerEntity.Price{Buy: priceDecimals, Sell: priceDecimals},
				}
			})

			data := findroute.NewFinderData(context.Background(), tokenByAddress, priceUSDByAddress, priceByAddress, &types.FindRouteState{
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
				// should have the expected number of pool along the path
				assert.Equal(t, tp.A.pools, tp.B.PoolAddresses)
				assert.Equal(t, tp.A.amountIn, tp.B.Input.Amount.Uint64())
				assert.Equal(t, tp.A.amountOut, tp.B.Output.Amount.Uint64())
			})
		}

		normalPriceUSD := lo.SliceToMap(tokenAddressList, func(adr string) (string, float64) { return adr, 1 })
		normalPriceUSD["gas"] = 20000000000
		normalPriceNative := lo.SliceToMap(tokenAddressList, func(adr string) (string, *big.Float) { return adr, big.NewFloat(100000000) })

		// use usd alone
		t.Run(fmt.Sprintf("%s - use USD price", tc.name), func(t *testing.T) { f(t, tc, normalPriceUSD, nil) })

		// if all tokens has the same Native price then the result should be the same
		t.Run(fmt.Sprintf("%s - use Native price", tc.name), func(t *testing.T) { f(t, tc, normalPriceUSD, normalPriceNative) })

		// if we're missing price for all or some tokens, then will fallback to compare amountOut
		// the result should be different (because now `pool-ab-1-highgas` is better than `pool-ab-1`)
		// but when comparing path we're still using usd, so should give the same result for now
		// will be split into another test later
		t.Run(fmt.Sprintf("%s - use Native price (none)", tc.name), func(t *testing.T) { f(t, tc, normalPriceUSD, map[string]*big.Float{}) })

		t.Run(fmt.Sprintf("%s - use Native price (some)", tc.name), func(t *testing.T) {
			f(t, tc, normalPriceUSD, map[string]*big.Float{
				"a": big.NewFloat(100000000),
				"c": big.NewFloat(100000000),
			})
		})

		// still need usd for native token (gas token)
		t.Run(fmt.Sprintf("%s - use Native price only", tc.name), func(t *testing.T) { f(t, tc, map[string]float64{"gas": 10000000000}, normalPriceNative) })
	}
}

type ThresholdSim struct {
	poolpkg.IPoolSimulator
	Threshold int64 // amountIn >= threshold: amountOut x2
}

func (s *ThresholdSim) CalcAmountOut(params poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	res, err := s.IPoolSimulator.CalcAmountOut(params)
	if err != nil {
		return nil, err
	}
	if params.TokenAmountIn.Amount.Int64() >= s.Threshold {
		res.TokenAmountOut.Amount.Mul(res.TokenAmountOut.Amount, big.NewInt(2))
		res.TokenAmountOut.AmountUsd *= 2
	}
	return res, nil
}
