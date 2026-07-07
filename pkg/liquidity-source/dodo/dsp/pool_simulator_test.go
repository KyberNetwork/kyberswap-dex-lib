package dsp

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/goccy/go-json"
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
	t.Parallel()
	pools := []string{
		// Pool data at block https://arbiscan.io/block/215792414

		// https://arbiscan.io/address/0xa6ec95be503f803bce9e7dd498602f1b28c9a02a#code
		"{\"address\":\"0xa6ec95be503f803bce9e7dd498602f1b28c9a02a\",\"swapFee\":100000000000000,\"exchange\":\"dodo-dsp\",\"type\":\"dodo-dsp\",\"timestamp\":1716870877,\"reserves\":[\"33336489800302\",\"1888512\"],\"tokens\":[{\"address\":\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\",\"name\":\"Wrapped Ether\",\"symbol\":\"WETH\",\"decimals\":18,\"weight\":50,\"swappable\":true},{\"address\":\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\",\"name\":\"Tether USD\",\"symbol\":\"USDT\",\"decimals\":6,\"weight\":50,\"swappable\":true}],\"extra\":\"{\\\"i\\\":\\\"3723935145\\\",\\\"K\\\":\\\"100000000000000\\\",\\\"B\\\":\\\"33336489800302\\\",\\\"Q\\\":\\\"1888512\\\",\\\"B0\\\":\\\"270192202826890\\\",\\\"Q0\\\":\\\"1005850\\\",\\\"R\\\":\\\"1\\\",\\\"mtFeeRate\\\":\\\"20000000000000\\\",\\\"lpFeeRate\\\":\\\"80000000000000\\\",\\\"swappable\\\":true}\",\"staticExtra\":\"{\\\"poolId\\\":\\\"0xa6ec95be503f803bce9e7dd498602f1b28c9a02a\\\",\\\"lpToken\\\":\\\"0xa6ec95be503f803bce9e7dd498602f1b28c9a02a\\\",\\\"type\\\":\\\"DSP\\\",\\\"tokens\\\":[\\\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\\\",\\\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\\\"],\\\"dodoV1SellHelper\\\":\\\"0xa5f36e822540efd11fcd77ec46626b916b217c3e\\\"}\"}",

		// https://arbiscan.io/address/0xd55128cbdba933bcf9b5f508108129ffe7e2e9bb#code
		"{\"address\":\"0xd55128cbdba933bcf9b5f508108129ffe7e2e9bb\",\"swapFee\":3000000000000000,\"exchange\":\"dodo-dsp\",\"type\":\"dodo-dsp\",\"timestamp\":1716870877,\"reserves\":[\"233467\",\"1117670600914973\"],\"tokens\":[{\"address\":\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\",\"name\":\"Tether USD\",\"symbol\":\"USDT\",\"decimals\":6,\"weight\":50,\"swappable\":true},{\"address\":\"0x0c1cf6883efa1b496b01f654e247b9b419873054\",\"name\":\"SushiSwap LP Token\",\"symbol\":\"SLP\",\"decimals\":18,\"weight\":50,\"swappable\":true}],\"extra\":\"{\\\"i\\\":\\\"538000000000000000000000000\\\",\\\"K\\\":\\\"100000000000000000\\\",\\\"B\\\":\\\"233467\\\",\\\"Q\\\":\\\"1117670600914973\\\",\\\"B0\\\":\\\"1036546\\\",\\\"Q0\\\":\\\"536995355922673\\\",\\\"R\\\":\\\"1\\\",\\\"mtFeeRate\\\":\\\"600000000000000\\\",\\\"lpFeeRate\\\":\\\"2400000000000000\\\",\\\"swappable\\\":true}\",\"staticExtra\":\"{\\\"poolId\\\":\\\"0xd55128cbdba933bcf9b5f508108129ffe7e2e9bb\\\",\\\"lpToken\\\":\\\"0xd55128cbdba933bcf9b5f508108129ffe7e2e9bb\\\",\\\"type\\\":\\\"DSP\\\",\\\"tokens\\\":[\\\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\\\",\\\"0x0c1cf6883efa1b496b01f654e247b9b419873054\\\"],\\\"dodoV1SellHelper\\\":\\\"0xa5f36e822540efd11fcd77ec46626b916b217c3e\\\"}\"}",
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
			bignumber.NewBig10("1888321"),
			bignumber.NewBig10("0"),
			bignumber.NewBig10("188"),
			nil,
		},

		// 0.1 WETH -> ? USDT
		{
			0,
			"0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
			"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9",
			bignumber.NewBig10("100000000000000000"),
			nil,
			nil,
			nil,
			libv2.ErrShouldNotBeZero,
		},

		// 10 USDT -> ? WETH
		{
			0,
			"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9",
			"0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
			bignumber.NewBig10("10000000"),
			bignumber.NewBig10("33330403871110"),
			bignumber.NewBig10("0"),
			bignumber.NewBig10("3333373723"),
			nil,
		},

		// 100 USDT -> ? WETH
		{
			0,
			"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9",
			"0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
			bignumber.NewBig10("100000000"),
			bignumber.NewBig10("33332883981369"),
			bignumber.NewBig10("0"),
			bignumber.NewBig10("3333621760"),
			nil,
		},

		// 10 USDT -> ? SLP
		{
			1,
			"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9",
			"0x0c1cf6883efa1b496b01f654e247b9b419873054",
			bignumber.NewBig10("10000000"),
			bignumber.NewBig10("1107962735102249"),
			bignumber.NewBig10("0"),
			bignumber.NewBig10("3333889874931"),
			nil,
		},

		// 100 USDT -> ? SLP
		{
			1,
			"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9",
			"0x0c1cf6883efa1b496b01f654e247b9b419873054",
			bignumber.NewBig10("100000000"),
			bignumber.NewBig10("1113774511602268"),
			bignumber.NewBig10("0"),
			bignumber.NewBig10("3351377667810"),
			nil,
		},

		// 10 SLP -> ? USDT
		{
			1,
			"0x0c1cf6883efa1b496b01f654e247b9b419873054",
			"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9",
			bignumber.NewBig10("10000000000000000000"),
			bignumber.NewBig10("232761"),
			bignumber.NewBig10("0"),
			bignumber.NewBig10("700"),
			nil,
		},

		// 100 SLP -> ? USDT
		{
			1,
			"0x0c1cf6883efa1b496b01f654e247b9b419873054",
			"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9",
			bignumber.NewBig10("100000000000000000000"),
			bignumber.NewBig10("232766"),
			bignumber.NewBig10("0"),
			bignumber.NewBig10("700"),
			nil,
		},

		// 100 SLP -> ? USDT
		{
			1,
			"0x0c1cf6883efa1b496b01f654e247b9b419873054",
			"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9",
			bignumber.NewBig10("1000000000000000000000"),
			nil,
			nil,
			nil,
			libv2.ErrShouldNotBeZero,
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
			amountOut, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
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
