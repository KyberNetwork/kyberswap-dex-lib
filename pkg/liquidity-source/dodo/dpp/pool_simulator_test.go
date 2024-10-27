package dpp

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dodo/libv2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	pools := []string{
		// Pool data at block https://arbiscan.io/block/215783877

		// https://arbiscan.io/address/0x8f11519f4f7c498e1f940b9de187d9c390321016#code
		"{\"address\":\"0x8f11519f4f7c498e1f940b9de187d9c390321016\",\"swapFee\":3000000000000000,\"exchange\":\"dodo-dpp\",\"type\":\"dodo-dpp\",\"timestamp\":1716868655,\"reserves\":[\"5682349893627314\",\"18472539\"],\"tokens\":[{\"address\":\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\",\"name\":\"Wrapped Ether\",\"symbol\":\"WETH\",\"decimals\":18,\"weight\":50,\"swappable\":true},{\"address\":\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\",\"name\":\"Tether USD\",\"symbol\":\"USDT\",\"decimals\":6,\"weight\":50,\"swappable\":true}],\"extra\":\"{\\\"i\\\":\\\"1200000000\\\",\\\"K\\\":\\\"1000000000000000000\\\",\\\"B\\\":\\\"5682349893627314\\\",\\\"Q\\\":\\\"18472539\\\",\\\"B0\\\":\\\"10116304445839343\\\",\\\"Q0\\\":\\\"9000000\\\",\\\"R\\\":\\\"1\\\",\\\"mtFeeRate\\\":\\\"0\\\",\\\"lpFeeRate\\\":\\\"3000000000000000\\\",\\\"swappable\\\":true}\",\"staticExtra\":\"{\\\"poolId\\\":\\\"0x8f11519f4f7c498e1f940b9de187d9c390321016\\\",\\\"lpToken\\\":\\\"\\\",\\\"type\\\":\\\"DPP\\\",\\\"tokens\\\":[\\\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\\\",\\\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\\\"],\\\"dodoV1SellHelper\\\":\\\"0xa5f36e822540efd11fcd77ec46626b916b217c3e\\\"}\"}",

		// https://arbiscan.io/address/0xb7392c0d85676de049121771c1edb31edd446336#code
		"{\"address\":\"0xb7392c0d85676de049121771c1edb31edd446336\",\"swapFee\":500000000000000,\"exchange\":\"dodo-dpp\",\"type\":\"dodo-dpp\",\"timestamp\":1716868655,\"reserves\":[\"900000000000000000\",\"100000\"],\"tokens\":[{\"address\":\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\",\"name\":\"Magic Internet Money\",\"symbol\":\"MIM\",\"decimals\":18,\"weight\":50,\"swappable\":true},{\"address\":\"0xaf88d065e77c8cc2239327c5edb3a432268e5831\",\"name\":\"USD Coin\",\"symbol\":\"USDC\",\"decimals\":6,\"weight\":50,\"swappable\":true}],\"extra\":\"{\\\"i\\\":\\\"1000000\\\",\\\"K\\\":\\\"250000000000000\\\",\\\"B\\\":\\\"900000000000000000\\\",\\\"Q\\\":\\\"100000\\\",\\\"B0\\\":\\\"900000000000000000\\\",\\\"Q0\\\":\\\"100000\\\",\\\"R\\\":\\\"0\\\",\\\"mtFeeRate\\\":\\\"0\\\",\\\"lpFeeRate\\\":\\\"500000000000000\\\",\\\"swappable\\\":true}\",\"staticExtra\":\"{\\\"poolId\\\":\\\"0xb7392c0d85676de049121771c1edb31edd446336\\\",\\\"lpToken\\\":\\\"\\\",\\\"type\\\":\\\"DPP\\\",\\\"tokens\\\":[\\\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\\\",\\\"0xaf88d065e77c8cc2239327c5edb3a432268e5831\\\"],\\\"dodoV1SellHelper\\\":\\\"0xa5f36e822540efd11fcd77ec46626b916b217c3e\\\"}\"}",
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
		// 0.01 WETH -> ? USDT
		{
			0,
			"0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
			"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9",
			bignumber.NewBig10("10000000000000000"),
			bignumber.NewBig10("13266558"),
			bignumber.NewBig10("0"),
			bignumber.NewBig10("39919"),
			nil,
		},

		// 0.1 WETH -> ? USDT
		{
			0,
			"0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
			"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9",
			bignumber.NewBig10("100000000000000000"),
			bignumber.NewBig10("17764167"),
			bignumber.NewBig10("0"),
			bignumber.NewBig10("53452"),
			nil,
		},

		// 10 USDT -> ? WETH
		{
			0,
			"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9",
			"0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
			bignumber.NewBig10("10000000"),
			bignumber.NewBig10("1792130882496387"),
			bignumber.NewBig10("0"),
			bignumber.NewBig10("5392570358564"),
			nil,
		},

		// 100 USDT -> ? WETH
		{
			0,
			"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9",
			"0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
			bignumber.NewBig10("100000000"),
			bignumber.NewBig10("4658502436846346"),
			bignumber.NewBig10("0"),
			bignumber.NewBig10("14017559990510"),
			nil,
		},

		// 10 MIM -> ? USDC
		{
			1,
			"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a",
			"0xaf88d065e77c8cc2239327c5edb3a432268e5831",
			bignumber.NewBig10("10000000000000000000"),
			nil,
			nil,
			nil,
			libv2.ErrShouldNotBeZero,
		},

		// 10 USDC -> ? MIM
		{
			1,
			"0xaf88d065e77c8cc2239327c5edb3a432268e5831",
			"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a",
			bignumber.NewBig10("10000000"),
			bignumber.NewBig10("899527759533293638"),
			bignumber.NewBig10("0"),
			bignumber.NewBig10("449988874203748"),
			nil,
		},

		// 100 USDC -> ? MIM
		{
			1,
			"0xaf88d065e77c8cc2239327c5edb3a432268e5831",
			"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a",
			bignumber.NewBig10("100000000"),
			bignumber.NewBig10("899547957640496812"),
			bignumber.NewBig10("0"),
			bignumber.NewBig10("449998978309403"),
			nil,
		},

		// 1000 USDC -> ? MIM
		{
			1,
			"0xaf88d065e77c8cc2239327c5edb3a432268e5831",
			"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a",
			bignumber.NewBig10("1000000000"),
			bignumber.NewBig10("899549797419018319"),
			bignumber.NewBig10("0"),
			bignumber.NewBig10("449999898658838"),
			nil,
		},
	}

	sims := lo.Map(pools, func(poolRedis string, _ int) *PoolSimulator {
		var poolEntity entity.Pool
		err := sonic.Unmarshal([]byte(poolRedis), &poolEntity)
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
