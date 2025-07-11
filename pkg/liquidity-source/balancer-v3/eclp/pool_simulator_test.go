package eclp

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/base"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/vault"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func newPool(poolStr string) *base.PoolSimulator {
	var poolE entity.Pool
	_ = json.Unmarshal([]byte(poolStr), &poolE)
	poolSim, _ := NewPoolSimulator(poolE)
	return poolSim
}

func pool1() *base.PoolSimulator {
	return newPool(`{"address":"0x5d7f2aac9999950f6ffb03394be584e1410bcfaf","exchange":"balancer-v3-eclp","type":"balancer-v3-eclp","timestamp":1743666215,"reserves":["7112661012533552","4570881"],"tokens":[{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","symbol":"WETH","decimals":18,"swappable":true},{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"hook\":{},\"fee\":\"1000000000000000\",\"aggrFee\":\"500000000000000000\",\"balsE18\":[\"7416589241266276\",\"5143281691971181138\"],\"decs\":[\"1\",\"1000000000000\"],\"rates\":[\"1042730593823768440\",\"1125227651293302350\"],\"buffs\":[{\"rate\":\"1042730593823768439\"},{\"rate\":\"1125227651214215093\"}],\"eclp\":{\"p\":{\"a\":\"1550000000000000000000\",\"b\":\"2900000000000000000000\",\"c\":\"476190422200635\",\"s\":\"999999886621334475\",\"l\":\"6000000000000000000000\"},\"d\":{\"tA\":{\"x\":\"-71194417720710388791873272380661517967\",\"y\":\"70223606325857393780377068191710603749\"},\"tB\":{\"x\":\"61901682449602783283884409155259788043\",\"y\":\"78537772504117652540633453925274067422\"},\"u\":\"63379080947523002588928208779431795\",\"v\":\"78537770618819626952384805221876672653\",\"w\":\"3959125853791535734173172101662641\",\"z\":\"-71194387540195651900451013855438212676\",\"DSq\":\"100000000000000000034081090601792885000\"}}}","staticExtra":"{\"buffs\":[\"0x0bfc9d54fc184518a81162f8fb99c2eaca081202\",\"0xd4fa2d31b7968e448877f69a96de69f5de8cd23e\"]}","blockNumber":22186972}`)
}

func pool2() *base.PoolSimulator {
	return newPool(`{"address":"0x6adda8a09dac8336abf13e70ee8bba1e55c3015f","exchange":"balancer-v3-eclp","type":"balancer-v3-eclp","timestamp":1751868430,"reserves":["5430800174919153651","2791778907"],"tokens":[{"address":"0x82af49447d8a07e3bd95bd0d56f35241523fbab1","symbol":"WETH","decimals":18,"swappable":true},{"address":"0xaf88d065e77c8cc2239327c5edb3a432268e5831","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"hook\":{},\"fee\":\"2000000000000000\",\"aggrFee\":\"500000000000000000\",\"balsE18\":[\"5682926679552568080\",\"3172111633565964967840\"],\"decs\":[\"1\",\"1000000000000\"],\"rates\":[\"1046425295815117658\",\"1136233111301304408\"],\"buffs\":[{\"rate\":\"1046425295815117656\"},{\"rate\":\"1136233110600910528\"}],\"eclp\":{\"p\":{\"a\":\"2300000000000000000000\",\"b\":\"3450000000000000000000\",\"c\":\"340136034746169\",\"s\":\"999999942153737260\",\"l\":\"15000000000000000000000\"},\"d\":{\"tA\":{\"x\":\"-81754800040802866166495335404043732617\",\"y\":\"57586045794865440326374685805400997563\"},\"tB\":{\"x\":\"60215136492456391650835048577140986617\",\"y\":\"79838194726552068708694627685354335017\"},\"u\":\"48289088472244617795808663907556578\",\"v\":\"79838192152144835849261396367525693446\",\"w\":\"7568757264380743767847482473451543\",\"z\":\"-81754783615942841514788990471890354787\",\"DSq\":\"99999999999999999903719824937248416100\"}}}","staticExtra":"{\"buffs\":[\"0x4ce13a79f45c1be00bdabd38b764ac28c082704e\",\"0x7f6501d3b98ee91f9b9535e4b0ac710fb0f9e0bc\"]}","blockNumber":355102143}`)
}

func poolAaveBuffered() *base.PoolSimulator {
	return newPool("{\"address\":\"0x6adda8a09dac8336abf13e70ee8bba1e55c3015f\",\"exchange\":\"balancer-v3-eclp\",\"type\":\"balancer-v3-eclp\",\"timestamp\":1752198134,\"reserves\":[\"2487343948055235912\",\"9964695470\"],\"tokens\":[{\"address\":\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\",\"symbol\":\"WETH\",\"decimals\":18,\"swappable\":true},{\"address\":\"0xaf88d065e77c8cc2239327c5edb3a432268e5831\",\"symbol\":\"USDC\",\"decimals\":6,\"swappable\":true}],\"extra\":\"{\\\"hook\\\":{},\\\"fee\\\":\\\"2000000000000000\\\",\\\"aggrFee\\\":\\\"500000000000000000\\\",\\\"balsE18\\\":[\\\"2603210576142303686\\\",\\\"11326420907101066582699\\\"],\\\"decs\\\":[\\\"1\\\",\\\"1000000000000\\\"],\\\"rates\\\":[\\\"1046582471305450013\\\",\\\"1136654997756902508\\\"],\\\"buffs\\\":[{\\\"rate\\\":\\\"1046582471305450013\\\"},{\\\"rate\\\":\\\"1136654997756902508\\\"}],\\\"eclp\\\":{\\\"p\\\":{\\\"a\\\":\\\"2300000000000000000000\\\",\\\"b\\\":\\\"3450000000000000000000\\\",\\\"c\\\":\\\"340136034746169\\\",\\\"s\\\":\\\"999999942153737260\\\",\\\"l\\\":\\\"15000000000000000000000\\\"},\\\"d\\\":{\\\"tA\\\":{\\\"x\\\":\\\"-81754800040802866166495335404043732617\\\",\\\"y\\\":\\\"57586045794865440326374685805400997563\\\"},\\\"tB\\\":{\\\"x\\\":\\\"60215136492456391650835048577140986617\\\",\\\"y\\\":\\\"79838194726552068708694627685354335017\\\"},\\\"u\\\":\\\"48289088472244617795808663907556578\\\",\\\"v\\\":\\\"79838192152144835849261396367525693446\\\",\\\"w\\\":\\\"7568757264380743767847482473451543\\\",\\\"z\\\":\\\"-81754783615942841514788990471890354787\\\",\\\"DSq\\\":\\\"99999999999999999903719824937248416100\\\"}}}\",\"staticExtra\":\"{\\\"buffs\\\":[\\\"0x4ce13a79f45c1be00bdabd38b764ac28c082704e\\\",\\\"0x7f6501d3b98ee91f9b9535e4b0ac710fb0f9e0bc\\\"]}\",\"blockNumber\":356416181}")
}

func TestCalcAmountOut(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name              string
		pool              *base.PoolSimulator
		tokenAmountIn     pool.TokenAmount
		tokenOut          string
		expectedAmountOut string
		expectedError     error
	}{
		{
			name: "1. 0->1 0x5d7f2aac9999950f6ffb03394be584e1410bcfaf",
			pool: pool1(),
			tokenAmountIn: pool.TokenAmount{
				Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				Amount: big.NewInt(1e14),
			},
			tokenOut:          "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			expectedAmountOut: "182071", // 182068
			expectedError:     nil,
		},
		{
			name: "1. 0->1 AmountIn is too small",
			pool: pool1(),
			tokenAmountIn: pool.TokenAmount{
				Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				Amount: big.NewInt(1000000),
			},
			tokenOut:          "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			expectedAmountOut: "0",
			expectedError:     vault.ErrAmountInTooSmall,
		},
		{
			name: "1. 0->1 AmountIn is too small for buffering",
			pool: pool1(),
			tokenAmountIn: pool.TokenAmount{
				Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				Amount: big.NewInt(1000),
			},
			tokenOut:          "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			expectedAmountOut: "0",
			expectedError:     shared.ErrWrapAmountTooSmall,
		},
		{
			name: "1. 0->1 ErrAssetBoundsExceeded",
			pool: pool1(),
			tokenAmountIn: pool.TokenAmount{
				Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				Amount: big.NewInt(1e17),
			},
			tokenOut:          "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			expectedAmountOut: "0",
			expectedError:     math.ErrAssetBoundsExceeded,
		},
		{
			name: "2. 0->1 ok",
			pool: pool2(),
			tokenAmountIn: pool.TokenAmount{
				Token:  "0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
				Amount: big.NewInt(28580146123451234),
			},
			tokenOut:          "0xaf88d065e77c8cc2239327c5edb3a432268e5831",
			expectedAmountOut: "73258603", // actual simulation gives "72145424" due to buffer imbalance
			expectedError:     nil,
		},
		{
			name: "aave. 0->1 ok",
			pool: poolAaveBuffered(),
			tokenAmountIn: pool.TokenAmount{
				Token:  "0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
				Amount: big.NewInt(100000000000000),
			},
			tokenOut:          "0xaf88d065e77c8cc2239327c5edb3a432268e5831",
			expectedAmountOut: "293761", // 293759
			expectedError:     nil,
		},
		{
			name: "aave. 1->0 ok",
			pool: poolAaveBuffered(),
			tokenAmountIn: pool.TokenAmount{
				Token:  "0xaf88d065e77c8cc2239327c5edb3a432268e5831",
				Amount: big.NewInt(100000000),
			},
			tokenOut:          "0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
			expectedAmountOut: "33880901758804406", // 33846106954701616
			expectedError:     nil,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return tc.pool.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: tc.tokenAmountIn,
					TokenOut:      tc.tokenOut,
				})
			})

			assert.Equal(t, tc.expectedError, err)
			if err == nil {
				assert.Equal(t, tc.expectedAmountOut, result.TokenAmountOut.Amount.String())
			}
		})
	}
}

func TestPoolSimulator_CalcAmountIn(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name             string
		pool             *base.PoolSimulator
		tokenAmountOut   pool.TokenAmount
		tokenIn          string
		expectedAmountIn string
		expectedError    error
	}{
		{
			name: "1. 0->1",
			pool: pool1(),
			tokenAmountOut: pool.TokenAmount{
				Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				Amount: big.NewInt(182071),
			},
			tokenIn:          "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			expectedAmountIn: "99998114793038",
			expectedError:    nil,
		},
	}
	for _, tc := range testcases[1:] {
		t.Run(tc.name, func(t *testing.T) {
			result, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountInResult, error) {
				return tc.pool.CalcAmountIn(pool.CalcAmountInParams{
					TokenAmountOut: tc.tokenAmountOut,
					TokenIn:        tc.tokenIn,
				})
			})

			assert.Equal(t, tc.expectedError, err)
			if err == nil {
				assert.Equal(t, tc.expectedAmountIn, result.TokenAmountIn.Amount.String())
			}
		})
	}
}
