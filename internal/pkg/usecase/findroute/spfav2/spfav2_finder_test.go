package spfav2

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"testing"

	composablestable "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/composable-stable"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/holiman/uint256"
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

			data := findroute.NewFinderData(context.Background(), tokenByAddress, priceUSDByAddress, nil, &types.FindRouteState{
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

type balancerPool struct {
	name                              string
	address                           string
	tokens                            []*entity.PoolToken
	reserves                          []string
	scalingFactors                    []string
	rateProviders                     []string
	swapFeePercentage                 uint64
	bptIndex                          int
	amp                               uint64
	isTokenExemptFromYieldProtocolFee []bool
	tokenRateCaches                   []composablestable.TokenRateCache
	protocolFeePercentageCache        map[int]*uint256.Int
	lastJoinExit                      composablestable.LastJoinExitData
	bptTotalSupply                    string
}

type balancerTestPaths []balancerTestSwap

type balancerTestSwap struct {
	poolName string
	// tokenIn, tokenOut
	tokens []string
}

var (
	data              findroute.FinderData
	poolAddressByName map[string]string
)

var balancerTokenByAddress = map[string]*entity.Token{
	"w1":  {Address: "w1"},
	"w2":  {Address: "w2"},
	"w3":  {Address: "w3"},
	"w4":  {Address: "w4"},
	"a":   {Address: "a"},
	"b":   {Address: "b"},
	"c":   {Address: "c"},
	"d":   {Address: "d"},
	"f":   {Address: "f"},
	"g":   {Address: "g"},
	"gas": {Address: "gas"},
}

var priceUSDByAddress = lo.MapValues(balancerTokenByAddress, func(v *entity.Token, key string) float64 { return 1 })

func initBalancerPools() findroute.FinderData {
	/**  pool format: balancer-tokens-bptIndex, pbtIndex means only swap token with index < pbtIndex or index - 1
	 *	 pool1: w1w2ab
	 *	 pool2: w3w4bcd
	 *	 pool3: afg
	 *	 all tokens started with w is whitelist token
	**/
	pools := []balancerPool{
		{
			// pool name will be balancer-w1w2ab-0, but according to composable balancer v2, pool Address must be unique with token at bpt index
			"balancer-w1w2ab-0",
			"w1",
			[]*entity.PoolToken{
				{Address: "w1"}, {Address: "w2"}, {Address: "a"}, {Address: "b"},
			},
			[]string{"2596148432157077319352279762223175", "40898799479796189246", "96043801875260816584", "29663011490936802030"},
			[]string{"1000000000000000000", "1000000009603216581", "1000000000137236044", "1000000000000000000"},
			[]string{"0x0000000000000000000000000000000000000000", "w2", "a", "b"},
			1000000000000,
			0,
			4000000,
			[]bool{false, false, false, false},
			[]composablestable.TokenRateCache{
				{},
				{
					Rate:     uint256.MustFromDecimal("1000000009603216581"),
					OldRate:  uint256.MustFromDecimal("1000000009603216581"),
					Duration: uint256.NewInt(21600),
					Expires:  uint256.NewInt(1692724859),
				},
				{
					Rate:     uint256.MustFromDecimal("1000000000137236044"),
					OldRate:  uint256.MustFromDecimal("1000000000137236044"),
					Duration: uint256.NewInt(21600),
					Expires:  uint256.NewInt(1692724859),
				},
				{
					Rate:     uint256.MustFromDecimal("1000000000000000000"),
					OldRate:  uint256.MustFromDecimal("1000000000000000000"),
					Duration: uint256.NewInt(21600),
					Expires:  uint256.NewInt(1692724859),
				},
			},
			map[int]*uint256.Int{0: uint256.NewInt(0), 2: uint256.NewInt(0)},
			composablestable.LastJoinExitData{
				LastJoinExitAmplification: uint256.NewInt(4000000),
				LastPostJoinExitInvariant: uint256.MustFromDecimal("4101410955893225440478258"),
			},
			"2596148432157243916047650351325428",
		},
		{
			name:    "balancer-w3w4bcd-0",
			address: "w3",
			tokens: []*entity.PoolToken{
				{Address: "w3"}, {Address: "w4"}, {Address: "b"}, {Address: "c"}, {Address: "d"},
			},
			reserves:                          []string{"3560721061507068661", "1315000334745039328", "25961484291276274900964501", "2636305", "44886"},
			scalingFactors:                    []string{"999000000000000000", "1001000000000000000", "1000000000000000000", "1001000000000000000000000000000", "100000000000000000000000000000"},
			rateProviders:                     []string{"0x47b584e4c7c4a030060450ec9e51d52d919b1fcb", "0x1a867225414678c2c6faf54b1123dcf86e09cae7", "0x0000000000000000000000000000000000000000", "0xbacd5f6a91e8d040f2989af877ac06b5a26f1c85", "0xf8bbc8c2ced9c24992b6ec9bbd0eddaf3bce70eb"},
			swapFeePercentage:                 1000000000000,
			bptIndex:                          0,
			amp:                               5000000,
			isTokenExemptFromYieldProtocolFee: []bool{false, false, false, false, false},
			tokenRateCaches: []composablestable.TokenRateCache{
				{
					Rate:     uint256.MustFromDecimal("999000000000000000"),
					OldRate:  uint256.MustFromDecimal("999000000000000000"),
					Duration: uint256.NewInt(0),
					Expires:  uint256.NewInt(1669237367),
				},
				{
					Rate:     uint256.MustFromDecimal("1001000000000000000"),
					OldRate:  uint256.MustFromDecimal("1001000000000000000"),
					Duration: uint256.NewInt(0),
					Expires:  uint256.NewInt(1669237367),
				},
				{},
				{
					Rate:     uint256.MustFromDecimal("1001000000000000000"),
					OldRate:  uint256.MustFromDecimal("1001000000000000000"),
					Duration: uint256.NewInt(0),
					Expires:  uint256.NewInt(1669237367),
				},
				{
					Rate:     uint256.MustFromDecimal("100000000000000000"),
					OldRate:  uint256.MustFromDecimal("100000000000000000"),
					Duration: uint256.NewInt(0),
					Expires:  uint256.NewInt(1669237367),
				},
			},
			protocolFeePercentageCache: map[int]*uint256.Int{0: uint256.NewInt(0), 2: uint256.NewInt(0)},
			lastJoinExit: composablestable.LastJoinExitData{
				LastJoinExitAmplification: uint256.NewInt(5000000),
				LastPostJoinExitInvariant: uint256.MustFromDecimal("7437153944755193005"),
			},
			bptTotalSupply: "2596148429127641468731722610694070",
		},
		{
			name:    "balancer-w2w3w4-2",
			address: "w4",
			tokens: []*entity.PoolToken{
				{Address: "w2"}, {Address: "w3"}, {Address: "w4"},
			},
			reserves:                          []string{"3690000000000000000", "6156138011132578142", "2596148429267403832116683817120311"},
			scalingFactors:                    []string{"1000000000000000000", "1000000000000000000", "1000000000000000000"},
			rateProviders:                     []string{"0xc7177b6e18c1abd725f5b75792e5f7a3ba5dbc2c", "0x0000000000000000000000000000000000000000", "0x0000000000000000000000000000000000000000"},
			swapFeePercentage:                 100000000000000,
			bptIndex:                          2,
			amp:                               200000,
			isTokenExemptFromYieldProtocolFee: []bool{false, false, false},
			tokenRateCaches: []composablestable.TokenRateCache{
				{
					Rate:     uint256.MustFromDecimal("1066831390111251946"),
					OldRate:  uint256.MustFromDecimal("1040114539407281434"),
					Duration: uint256.NewInt(10800),
					Expires:  uint256.NewInt(1711706837),
				},
				{},
				{},
			},
			protocolFeePercentageCache: map[int]*uint256.Int{0: uint256.NewInt(0), 2: uint256.NewInt(0)},
			lastJoinExit: composablestable.LastJoinExitData{
				LastJoinExitAmplification: uint256.NewInt(200000),
				LastPostJoinExitInvariant: uint256.MustFromDecimal("9982148564347489737"),
			},
			bptTotalSupply: "2596148429267413814265248164610048",
		},
		{
			"balancer-afg-2",
			"g",
			[]*entity.PoolToken{
				{Address: "a"}, {Address: "f"}, {Address: "g"},
			},
			[]string{"6000000000000000000", "6000000000000000000", "8000000000000000000"},
			[]string{"1000000000000000000", "1000000000000000000", "1000000000000000000"},
			[]string{"0x0000000000000000000000000000000000000000", "a", "f"},
			100000000000000,
			2,
			1500000,
			[]bool{false, false, false, false},
			[]composablestable.TokenRateCache{
				{},
				{
					Rate:     uint256.MustFromDecimal("1000000009603216581"),
					OldRate:  uint256.MustFromDecimal("1000000009603216581"),
					Duration: uint256.NewInt(21600),
					Expires:  uint256.NewInt(1692724859),
				},
				{
					Rate:     uint256.MustFromDecimal("1000000000137236044"),
					OldRate:  uint256.MustFromDecimal("1000000000137236044"),
					Duration: uint256.NewInt(21600),
					Expires:  uint256.NewInt(1692724859),
				},
				{
					Rate:     uint256.MustFromDecimal("1000000000000000000"),
					OldRate:  uint256.MustFromDecimal("1000000000000000000"),
					Duration: uint256.NewInt(21600),
					Expires:  uint256.NewInt(1692724859),
				},
			},
			map[int]*uint256.Int{0: uint256.NewInt(0), 2: uint256.NewInt(0)},
			composablestable.LastJoinExitData{
				LastJoinExitAmplification: uint256.NewInt(4000000),
				LastPostJoinExitInvariant: uint256.MustFromDecimal("4101410955893225440478258"),
			},
			"2596148432157243916047650351325428",
		},
	}
	poolEntities := lo.Map(pools, func(p balancerPool, _ int) entity.Pool {
		factors := make([]*uint256.Int, 0, len(p.scalingFactors))
		for _, f := range p.scalingFactors {
			factors = append(factors, uint256.MustFromDecimal(f))
		}
		staticExtra := composablestable.StaticExtra{
			PoolID:   p.name,
			BptIndex: p.bptIndex,
		}
		staticExtraBytes, _ := json.Marshal(staticExtra)
		extra := composablestable.Extra{
			ScalingFactors:                    factors,
			Amp:                               uint256.NewInt(p.amp),
			SwapFeePercentage:                 uint256.NewInt(p.swapFeePercentage),
			BptTotalSupply:                    uint256.MustFromDecimal(p.bptTotalSupply),
			LastJoinExit:                      p.lastJoinExit,
			TokenRateCaches:                   p.tokenRateCaches,
			RateProviders:                     p.rateProviders,
			ProtocolFeePercentageCache:        p.protocolFeePercentageCache,
			IsTokenExemptFromYieldProtocolFee: p.isTokenExemptFromYieldProtocolFee,
			InRecoveryMode:                    true,
		}

		extraBytes, _ := json.Marshal(extra)
		return entity.Pool{
			Address:     p.address,
			Tokens:      p.tokens,
			Reserves:    p.reserves,
			StaticExtra: string(staticExtraBytes),
			Extra:       string(extraBytes),
		}
	})
	poolAddressByName = lo.SliceToMap(pools, func(pool balancerPool) (string, string) {
		return pool.name, pool.address
	})
	poolByAddress := lo.SliceToMap(poolEntities, func(poolEntity entity.Pool) (string, poolpkg.IPoolSimulator) {
		pool, _ := composablestable.NewPoolSimulator(poolEntity)
		return pool.GetAddress(), pool
	})
	return findroute.NewFinderData(context.Background(), balancerTokenByAddress, priceUSDByAddress, nil, &types.FindRouteState{
		Pools:     poolByAddress,
		SwapLimit: make(map[string]poolpkg.SwapLimit),
	})
}

// Using pools with multiple tokens as composable balancer
// Motivation for enforcing only whitelist tokens as hop token is avoiding FOT token as hop token and reduce router-get-route performance
// When a hop token is FOT token, calculated route will be wrong due to next path calculation is wrong.
func TestFindRoute_WithWhiteListToken(t *testing.T) {

	whitelistTokens := map[string]bool{"w1": true, "w2": true, "w3": true, "w4": true}

	finder := NewSPFAv2Finder(
		maxHop,
		whitelistTokens,
		distributionPercent,
		maxPathsInRoute,
		maxPathToGenerate,
		maxPathToReturn,
		minPartUSD,
		minThresholdAmountInUSD,
		float64(maxThresholdAmountInUSD),
		func(sourceHash uint64, tokenIn, tokenOut string) []*entity.MinimalPath { return nil },
	)

	testCases := []struct {
		name          string
		params        *types.AggregateParams
		expectedPaths []balancerTestPaths
		err           error
	}{
		{
			name: "There are only 4 valid routes (routes with max 3 hops) that swap from b to g: (b -> a -> f -> g) (b -> a -> g) (b -> w1 -> a -> g) (b -> w2 -> a -> g), but no route contains only whitelist tokens hope, expected result is nil",
			params: &types.AggregateParams{
				TokenIn:          *balancerTokenByAddress["b"],
				TokenOut:         *balancerTokenByAddress["g"],
				GasToken:         *balancerTokenByAddress["gas"],
				TokenInPriceUSD:  priceUSDByAddress["b"],
				TokenOutPriceUSD: priceUSDByAddress["g"],
				GasTokenPriceUSD: priceUSDByAddress["gas"],
				AmountIn:         big.NewInt(10000000000),
				Sources:          []string{},
				SaveGas:          false,
				GasInclude:       true,
				GasPrice:         big.NewFloat(1),
				ExtraFee:         valueobject.ZeroExtraFee,
			},
		},
		{
			name: "c can not swap to g because routes from c to g must go through a, a is not whitelist token",
			params: &types.AggregateParams{
				TokenIn:          *balancerTokenByAddress["c"],
				TokenOut:         *balancerTokenByAddress["g"],
				GasToken:         *balancerTokenByAddress["gas"],
				TokenInPriceUSD:  priceUSDByAddress["c"],
				TokenOutPriceUSD: priceUSDByAddress["g"],
				GasTokenPriceUSD: priceUSDByAddress["gas"],
				AmountIn:         big.NewInt(10000000000),
				Sources:          []string{},
				SaveGas:          false,
				GasInclude:       true,
				GasPrice:         big.NewFloat(1),
				ExtraFee:         valueobject.ZeroExtraFee,
			},
		},
		{
			name: "There are only 3 valid routes (routes with max 3 hops) that swap from b to f: (b -> a -> f) (b -> w1 -> a -> f) (b -> w2 -> a -> f), but no route contains only whitelist tokens hope, expected result is nil",
			params: &types.AggregateParams{
				TokenIn:          *balancerTokenByAddress["b"],
				TokenOut:         *balancerTokenByAddress["f"],
				GasToken:         *balancerTokenByAddress["gas"],
				TokenInPriceUSD:  priceUSDByAddress["b"],
				TokenOutPriceUSD: priceUSDByAddress["f"],
				GasTokenPriceUSD: priceUSDByAddress["gas"],
				AmountIn:         big.NewInt(10000000000),
				Sources:          []string{},
				SaveGas:          false,
				GasInclude:       true,
				GasPrice:         big.NewFloat(1),
				ExtraFee:         valueobject.ZeroExtraFee,
			},
		},
		{
			name: "There are 3 valid routes (paths with max 3 hops) that swap from b to a: (b -> a) (b -> w3 -> w2 -> a) (b -> w4 -> w2 -> a); b -> a yields 10004743045, b -> w3 yields 400207065150807045, can not swap from b -> w4 due to error",
			params: &types.AggregateParams{
				TokenIn:          *balancerTokenByAddress["b"],
				TokenOut:         *balancerTokenByAddress["a"],
				GasToken:         *balancerTokenByAddress["gas"],
				TokenInPriceUSD:  priceUSDByAddress["b"],
				TokenOutPriceUSD: priceUSDByAddress["a"],
				GasTokenPriceUSD: priceUSDByAddress["gas"],
				AmountIn:         big.NewInt(10000000000),
				Sources:          []string{},
				SaveGas:          false,
				GasInclude:       true,
				GasPrice:         big.NewFloat(1),
				ExtraFee:         valueobject.ZeroExtraFee,
			},
			expectedPaths: []balancerTestPaths{
				{
					{poolName: "balancer-w3w4bcd-0", tokens: []string{"b", "w3"}},
					{poolName: "balancer-w2w3w4-2", tokens: []string{"w3", "w2"}},
					{poolName: "balancer-w1w2ab-0", tokens: []string{"w2", "a"}},
				},
			},
		},
		{
			// swap c to w3 provide 10906330747951644901195478744631532, swap c to w4 provide 1314653885181311288
			name: "There are only 1 valid routes that swap from c to w1: (c -> w3 -> w2 -> w1), another swap is (c -> w4 -> w2 -> w1) is invalid",
			params: &types.AggregateParams{
				TokenIn:          *balancerTokenByAddress["c"],
				TokenOut:         *balancerTokenByAddress["w1"],
				GasToken:         *balancerTokenByAddress["gas"],
				TokenInPriceUSD:  priceUSDByAddress["c"],
				TokenOutPriceUSD: priceUSDByAddress["w1"],
				GasTokenPriceUSD: priceUSDByAddress["gas"],
				AmountIn:         big.NewInt(10000000000),
				Sources:          []string{},
				SaveGas:          false,
				GasInclude:       true,
				GasPrice:         big.NewFloat(1),
				ExtraFee:         valueobject.ZeroExtraFee,
			},
			expectedPaths: []balancerTestPaths{
				{
					{poolName: "balancer-w3w4bcd-0", tokens: []string{"c", "w3"}},
					{poolName: "balancer-w2w3w4-2", tokens: []string{"w3", "w2"}},
					{poolName: "balancer-w1w2ab-0", tokens: []string{"w2", "w1"}},
				},
			},
		},
		{
			name: "There is only 1 valid routes that swap from c to d: (c -> d), others must traverse to the same pool which is not acceptable",
			params: &types.AggregateParams{
				TokenIn:          *balancerTokenByAddress["c"],
				TokenOut:         *balancerTokenByAddress["d"],
				GasToken:         *balancerTokenByAddress["gas"],
				TokenInPriceUSD:  priceUSDByAddress["c"],
				TokenOutPriceUSD: priceUSDByAddress["d"],
				GasTokenPriceUSD: priceUSDByAddress["gas"],
				AmountIn:         big.NewInt(10000000000),
				Sources:          []string{},
				SaveGas:          false,
				GasInclude:       true,
				GasPrice:         big.NewFloat(1),
				ExtraFee:         valueobject.ZeroExtraFee,
			},
			expectedPaths: []balancerTestPaths{
				{
					{poolName: "balancer-w3w4bcd-0", tokens: []string{"c", "d"}},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data = initBalancerPools()
			input := findroute.Input{
				TokenInAddress:   tc.params.TokenIn.Address,
				TokenOutAddress:  tc.params.TokenOut.Address,
				AmountIn:         tc.params.AmountIn,
				GasPrice:         tc.params.GasPrice,
				GasTokenPriceUSD: tc.params.GasTokenPriceUSD,
				SaveGas:          tc.params.SaveGas,
				GasInclude:       tc.params.GasInclude,
			}
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
			for _, r := range allRoutes {
				for _, p := range r.Paths {
					fmt.Printf("tokens: ")
					for _, t := range p.Tokens {
						fmt.Printf("%s	", t.Address)
					}
				}
			}

			// first check number of possible paths
			require.Equal(t, len(tc.expectedPaths), len(routes.Paths))

			// then check each path
			lo.ForEach(lo.Zip2(tc.expectedPaths, routes.Paths), func(tp lo.Tuple2[balancerTestPaths, *valueobject.Path], _ int) {
				expectedPath := tp.A
				actualPath := tp.B.PoolAddresses
				expectedTokens := []string{}
				expectedTokens = append(expectedTokens, input.TokenInAddress)

				// should have the expected number of pool along the path
				require.Equal(t, len(expectedPath), len(actualPath))
				lo.ForEach(lo.Zip2(expectedPath, actualPath), func(tp lo.Tuple2[balancerTestSwap, string], _ int) {
					expectedPool := tp.A
					actualPool := tp.B

					assert.Equal(t, poolAddressByName[expectedPool.poolName], actualPool)
					expectedTokens = append(expectedTokens, expectedPool.tokens[1])
				})

				// should compare tokens because a pool contains many tokens
				actualTokens := tp.B.Tokens
				lo.ForEach(lo.Zip2(expectedTokens, actualTokens), func(tp lo.Tuple2[string, *entity.Token], _ int) {
					assert.Equal(t, tp.A, tp.B.Address)
				})
			})
		})
	}
}
