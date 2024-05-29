package classical

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	pools := []string{
		// Pool data at block https://arbiscan.io/block/214416932

		// https://arbiscan.io/address/0xb42a054d950dafd872808b3c839fbb7afb86e14c#readContract
		"{\"address\":\"0xb42a054d950dafd872808b3c839fbb7afb86e14c\",\"swapFee\":3000000000000000,\"exchange\":\"dodo-classical\",\"type\":\"dodo-classical\",\"timestamp\":1716521335,\"reserves\":[\"5293182\",\"10402621507\"],\"tokens\":[{\"address\":\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\",\"name\":\"Wrapped BTC\",\"symbol\":\"WBTC\",\"decimals\":8,\"weight\":50,\"swappable\":true},{\"address\":\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\",\"name\":\"USD Coin (Arb1)\",\"symbol\":\"USDC\",\"decimals\":6,\"weight\":50,\"swappable\":true}],\"extra\":\"{\\\"B\\\":\\\"5293182\\\",\\\"Q\\\":\\\"10402621507\\\",\\\"B0\\\":\\\"5313565\\\",\\\"Q0\\\":\\\"10388770142\\\",\\\"rStatus\\\":1,\\\"oraclePrice\\\":\\\"678741575565600000000\\\",\\\"k\\\":\\\"300000000000000000\\\",\\\"mtFeeRate\\\":\\\"600000000000000\\\",\\\"lpFeeRate\\\":\\\"2400000000000000\\\",\\\"swappable\\\":true}\",\"staticExtra\":\"{\\\"poolId\\\":\\\"0xb42a054d950dafd872808b3c839fbb7afb86e14c\\\",\\\"lpToken\\\":\\\"0xb94904bbe8a625709162dc172875fbc51c477abb\\\",\\\"type\\\":\\\"CLASSICAL\\\",\\\"tokens\\\":[\\\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\\\",\\\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\\\"],\\\"dodoV1SellHelper\\\":\\\"0xa5f36e822540efd11fcd77ec46626b916b217c3e\\\"}\"}",

		// https://arbiscan.io/address/0xe4b2dfc82977dd2dce7e8d37895a6a8f50cbb4fb
		"{\"address\":\"0xe4b2dfc82977dd2dce7e8d37895a6a8f50cbb4fb\",\"swapFee\":10000000000000,\"exchange\":\"dodo-classical\",\"type\":\"dodo-classical\",\"timestamp\":1716521335,\"reserves\":[\"1444873953831\",\"578850766374\"],\"tokens\":[{\"address\":\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\",\"name\":\"Tether USD\",\"symbol\":\"USDT\",\"decimals\":6,\"weight\":50,\"swappable\":true},{\"address\":\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\",\"name\":\"USD Coin (Arb1)\",\"symbol\":\"USDC\",\"decimals\":6,\"weight\":50,\"swappable\":true}],\"extra\":\"{\\\"B\\\":\\\"1444873953831\\\",\\\"Q\\\":\\\"578850766374\\\",\\\"B0\\\":\\\"978121462386\\\",\\\"Q0\\\":\\\"1045528008085\\\",\\\"rStatus\\\":2,\\\"oraclePrice\\\":\\\"1000000000000000000\\\",\\\"k\\\":\\\"200000000000000\\\",\\\"mtFeeRate\\\":\\\"10000000000000\\\",\\\"lpFeeRate\\\":\\\"0\\\",\\\"swappable\\\":true}\",\"staticExtra\":\"{\\\"poolId\\\":\\\"0xe4b2dfc82977dd2dce7e8d37895a6a8f50cbb4fb\\\",\\\"lpToken\\\":\\\"0x82b423848cdd98740fb57f961fa692739f991633\\\",\\\"type\\\":\\\"CLASSICAL\\\",\\\"tokens\\\":[\\\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\\\",\\\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\\\"],\\\"dodoV1SellHelper\\\":\\\"0xa5f36e822540efd11fcd77ec46626b916b217c3e\\\"}\"}",
	}

	testcases := []struct {
		poolIdx                   int
		tokenIn                   string
		tokenOut                  string
		amountIn                  *big.Int
		expectedAmountOut         *big.Int
		expectedRemainingAmountIn *big.Int
		expectedFee               *big.Int
		expectedErr               error
	}{
		// 0.01 WBTC -> ? USDC.e
		{
			0,
			"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f",
			"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			bignumber.NewBig10("1000000"),
			bignumber.NewBig10("663670872"),
			bignumber.NewBig10("0"),
			bignumber.NewBig10("1997002"),
			nil,
		},

		// 0.1 WBTC -> ? USDC.e
		{
			0,
			"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f",
			"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			bignumber.NewBig10("10000000"),
			bignumber.NewBig10("5203553797"),
			bignumber.NewBig10("0"),
			bignumber.NewBig10("15657633"),
			nil,
		},

		// 1 WBTC -> ? USDC.e
		{
			0,
			"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f",
			"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			bignumber.NewBig10("100000000"),
			bignumber.NewBig10("9867487281"),
			bignumber.NewBig10("0"),
			bignumber.NewBig10("29691536"),
			nil,
		},

		// 100 USDT -> ? USDC.e
		{
			1,
			"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9",
			"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			bignumber.NewBig10("100000000"),
			bignumber.NewBig10("99953762"),
			bignumber.NewBig10("0"),
			bignumber.NewBig10("999"),
			nil,
		},

		// 1000 USDT -> ? USDC.e
		{
			1,
			"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9",
			"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			bignumber.NewBig10("1000000000"),
			bignumber.NewBig10("999536601"),
			bignumber.NewBig10("0"),
			bignumber.NewBig10("9995"),
			nil,
		},

		// 10000 USDT -> ? USDC.e
		{
			1,
			"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9",
			"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			bignumber.NewBig10("10000000000"),
			bignumber.NewBig10("9995262737"),
			bignumber.NewBig10("0"),
			bignumber.NewBig10("99953"),
			nil,
		},
	}

	sims := lo.Map(pools, func(poolRedis string, _ int) *PoolSimulator {
		var poolEntity entity.Pool
		err := json.Unmarshal([]byte(poolRedis), &poolEntity)
		require.Nil(t, err)
		p, err := NewPoolSimulator(poolEntity)
		require.Nil(t, err)
		return p
	})

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx+1), func(t *testing.T) {
			p := sims[tc.poolIdx]
			amountOut, err := testutil.MustConcurrentSafe[*pool.CalcAmountOutResult](t, func() (any, error) {
				return p.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{
						Token:  tc.tokenIn,
						Amount: tc.amountIn,
					},
					TokenOut: tc.tokenOut,
					Limit:    nil,
				})
			})

			if err != nil {
				assert.ErrorIsf(t, err, tc.expectedErr, "expected error %v, got %v", tc.expectedErr, err)
				return
			}

			assert.Equalf(t, tc.tokenOut, amountOut.TokenAmountOut.Token, "expected token in %v, got %v", tc.tokenOut, amountOut.TokenAmountOut.Token)
			assert.Equalf(t, tc.expectedAmountOut, amountOut.TokenAmountOut.Amount, "expected amount in %v, got %v", tc.expectedAmountOut.String(), amountOut.TokenAmountOut.Amount.String())
			assert.Equalf(t, tc.expectedRemainingAmountIn.String(), amountOut.RemainingTokenAmountIn.Amount.String(), "expected remaining amount in %v, got %v", tc.expectedRemainingAmountIn.String(), amountOut.RemainingTokenAmountIn.Amount.String())
			assert.Equalf(t, tc.expectedFee, amountOut.Fee.Amount, "expected fee %v, got %v", tc.expectedFee, amountOut.Fee.Amount)
		})
	}
}

func TestPoolSimulator_CalcAmountIn(t *testing.T) {
	pools := []string{
		// Pool data at block https://arbiscan.io/block/214416932

		// https://arbiscan.io/address/0xb42a054d950dafd872808b3c839fbb7afb86e14c#readContract
		"{\"address\":\"0xb42a054d950dafd872808b3c839fbb7afb86e14c\",\"swapFee\":3000000000000000,\"exchange\":\"dodo-classical\",\"type\":\"dodo-classical\",\"timestamp\":1716521335,\"reserves\":[\"5293182\",\"10402621507\"],\"tokens\":[{\"address\":\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\",\"name\":\"Wrapped BTC\",\"symbol\":\"WBTC\",\"decimals\":8,\"weight\":50,\"swappable\":true},{\"address\":\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\",\"name\":\"USD Coin (Arb1)\",\"symbol\":\"USDC\",\"decimals\":6,\"weight\":50,\"swappable\":true}],\"extra\":\"{\\\"B\\\":\\\"5293182\\\",\\\"Q\\\":\\\"10402621507\\\",\\\"B0\\\":\\\"5313565\\\",\\\"Q0\\\":\\\"10388770142\\\",\\\"rStatus\\\":1,\\\"oraclePrice\\\":\\\"678741575565600000000\\\",\\\"k\\\":\\\"300000000000000000\\\",\\\"mtFeeRate\\\":\\\"600000000000000\\\",\\\"lpFeeRate\\\":\\\"2400000000000000\\\",\\\"swappable\\\":true}\",\"staticExtra\":\"{\\\"poolId\\\":\\\"0xb42a054d950dafd872808b3c839fbb7afb86e14c\\\",\\\"lpToken\\\":\\\"0xb94904bbe8a625709162dc172875fbc51c477abb\\\",\\\"type\\\":\\\"CLASSICAL\\\",\\\"tokens\\\":[\\\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\\\",\\\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\\\"],\\\"dodoV1SellHelper\\\":\\\"0xa5f36e822540efd11fcd77ec46626b916b217c3e\\\"}\"}",

		// https://arbiscan.io/address/0xe4b2dfc82977dd2dce7e8d37895a6a8f50cbb4fb
		"{\"address\":\"0xe4b2dfc82977dd2dce7e8d37895a6a8f50cbb4fb\",\"swapFee\":10000000000000,\"exchange\":\"dodo-classical\",\"type\":\"dodo-classical\",\"timestamp\":1716521335,\"reserves\":[\"1444873953831\",\"578850766374\"],\"tokens\":[{\"address\":\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\",\"name\":\"Tether USD\",\"symbol\":\"USDT\",\"decimals\":6,\"weight\":50,\"swappable\":true},{\"address\":\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\",\"name\":\"USD Coin (Arb1)\",\"symbol\":\"USDC\",\"decimals\":6,\"weight\":50,\"swappable\":true}],\"extra\":\"{\\\"B\\\":\\\"1444873953831\\\",\\\"Q\\\":\\\"578850766374\\\",\\\"B0\\\":\\\"978121462386\\\",\\\"Q0\\\":\\\"1045528008085\\\",\\\"rStatus\\\":2,\\\"oraclePrice\\\":\\\"1000000000000000000\\\",\\\"k\\\":\\\"200000000000000\\\",\\\"mtFeeRate\\\":\\\"10000000000000\\\",\\\"lpFeeRate\\\":\\\"0\\\",\\\"swappable\\\":true}\",\"staticExtra\":\"{\\\"poolId\\\":\\\"0xe4b2dfc82977dd2dce7e8d37895a6a8f50cbb4fb\\\",\\\"lpToken\\\":\\\"0x82b423848cdd98740fb57f961fa692739f991633\\\",\\\"type\\\":\\\"CLASSICAL\\\",\\\"tokens\\\":[\\\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\\\",\\\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\\\"],\\\"dodoV1SellHelper\\\":\\\"0xa5f36e822540efd11fcd77ec46626b916b217c3e\\\"}\"}",
	}

	testcases := []struct {
		poolIdx                    int
		tokenIn                    string
		tokenOut                   string
		amountOut                  *big.Int
		expectedAmountIn           *big.Int
		expectedRemainingAmountOut *big.Int
		expectedFee                *big.Int
		expectedErr                error
	}{
		// ? USDC.e -> 0.01 WBTC
		{
			0,
			"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f",
			bignumber.NewBig10("1000000"),
			bignumber.NewBig10("730469805"),
			bignumber.NewBig10("0"),
			bignumber.NewBig10("3000"),
			nil,
		},

		// ? USDC.e -> 0.05 WBTC
		{
			0,
			"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f",
			bignumber.NewBig10("5000000"),
			bignumber.NewBig10("21963175851"),
			bignumber.NewBig10("0"),
			bignumber.NewBig10("15000"),
			nil,
		},

		// ? USDC.e -> 0.1 WBTC
		{
			0,
			"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f",
			bignumber.NewBig10("10000000"),
			nil,
			nil,
			nil,
			ErrBaseBalanceNotEnough,
		},

		// ? USDC.e -> 100 USDT
		{
			1,
			"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9",
			bignumber.NewBig10("100000000"),
			bignumber.NewBig10("99955782"),
			bignumber.NewBig10("0"),
			bignumber.NewBig10("1000"),
			nil,
		},

		// ? USDC.e -> 1000 USDT
		{
			1,
			"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9",
			bignumber.NewBig10("1000000000"),
			bignumber.NewBig10("999558842"),
			bignumber.NewBig10("0"),
			bignumber.NewBig10("10000"),
			nil,
		},

		// ? USDC.e -> 10000 USDT
		{
			1,
			"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9",
			bignumber.NewBig10("10000000000"),
			bignumber.NewBig10("9995687848"),
			bignumber.NewBig10("0"),
			bignumber.NewBig10("100000"),
			nil,
		},

		// ? USDC.e -> 100000000 USDT
		{
			1,
			"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9",
			bignumber.NewBig10("100000000000000"),
			nil,
			nil,
			nil,
			ErrBaseBalanceNotEnough,
		},
	}

	sims := lo.Map(pools, func(poolRedis string, _ int) *PoolSimulator {
		var poolEntity entity.Pool
		err := json.Unmarshal([]byte(poolRedis), &poolEntity)
		require.Nil(t, err)
		p, err := NewPoolSimulator(poolEntity)
		require.Nil(t, err)
		return p
	})

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx+1), func(t *testing.T) {
			p := sims[tc.poolIdx]
			amountIn, err := testutil.MustConcurrentSafe[*pool.CalcAmountInResult](t, func() (any, error) {
				return p.CalcAmountIn(pool.CalcAmountInParams{
					TokenAmountOut: pool.TokenAmount{
						Token:  tc.tokenOut,
						Amount: tc.amountOut,
					},
					TokenIn: tc.tokenIn,
					Limit:   nil,
				})
			})

			if err != nil {
				assert.ErrorIsf(t, err, tc.expectedErr, "expected error %v, got %v", tc.expectedErr, err)
				return
			}

			assert.Equalf(t, tc.tokenIn, amountIn.TokenAmountIn.Token, "expected token in %v, got %v", tc.tokenIn, amountIn.TokenAmountIn.Token)
			assert.Equalf(t, tc.expectedAmountIn, amountIn.TokenAmountIn.Amount, "expected amount in %v, got %v", tc.expectedAmountIn.String(), amountIn.TokenAmountIn.Amount.String())
			assert.Equalf(t, tc.expectedRemainingAmountOut.String(), amountIn.RemainingTokenAmountOut.Amount.String(), "expected remaining amount in %v, got %v", tc.expectedRemainingAmountOut.String(), amountIn.RemainingTokenAmountOut.Amount.String())
			assert.Equalf(t, tc.expectedFee, amountIn.Fee.Amount, "expected fee %v, got %v", tc.expectedFee, amountIn.Fee.Amount)
		})
	}
}
