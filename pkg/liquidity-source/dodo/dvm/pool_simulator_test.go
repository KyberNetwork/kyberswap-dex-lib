package dvm

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
		// Pool data at block https://arbiscan.io/block/215765443

		// https://arbiscan.io/address/0xb627b318a537dff3883fcb7f0bd247ab6201b8d3#code
		"{\"address\":\"0xb627b318a537dff3883fcb7f0bd247ab6201b8d3\",\"swapFee\":100000000000000,\"exchange\":\"dodo-dvm\",\"type\":\"dodo-dvm\",\"timestamp\":1716863956,\"reserves\":[\"1001\",\"0\"],\"tokens\":[{\"address\":\"0x5330467941b3691a2c838769a58ddc5fca22ddec\",\"name\":\"BERD\",\"symbol\":\"BERD\",\"decimals\":18,\"weight\":50,\"swappable\":true},{\"address\":\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\",\"name\":\"Wrapped Ether\",\"symbol\":\"WETH\",\"decimals\":18,\"weight\":50,\"swappable\":true}],\"extra\":\"{\\\"i\\\":\\\"10000000\\\",\\\"K\\\":\\\"500000000000000000\\\",\\\"B\\\":\\\"1001\\\",\\\"Q\\\":\\\"0\\\",\\\"B0\\\":\\\"1001\\\",\\\"Q0\\\":\\\"0\\\",\\\"R\\\":\\\"1\\\",\\\"mtFeeRate\\\":\\\"20000000000000\\\",\\\"lpFeeRate\\\":\\\"80000000000000\\\",\\\"swappable\\\":true}\",\"staticExtra\":\"{\\\"poolId\\\":\\\"0xb627b318a537dff3883fcb7f0bd247ab6201b8d3\\\",\\\"lpToken\\\":\\\"0xb627b318a537dff3883fcb7f0bd247ab6201b8d3\\\",\\\"type\\\":\\\"DVM\\\",\\\"tokens\\\":[\\\"0x5330467941b3691a2c838769a58ddc5fca22ddec\\\",\\\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\\\"],\\\"dodoV1SellHelper\\\":\\\"0xa5f36e822540efd11fcd77ec46626b916b217c3e\\\"}\"}",

		// https://arbiscan.io/address/0x68276dc302d390245f3382eb4d2ea3a9317d46ef#code
		"{\"address\":\"0x68276dc302d390245f3382eb4d2ea3a9317d46ef\",\"swapFee\":3000000000000000,\"exchange\":\"dodo-dvm\",\"type\":\"dodo-dvm\",\"timestamp\":1716863956,\"reserves\":[\"15580539464573\",\"54845488636364795\"],\"tokens\":[{\"address\":\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\",\"name\":\"Wrapped Ether\",\"symbol\":\"WETH\",\"decimals\":18,\"weight\":50,\"swappable\":true},{\"address\":\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\",\"name\":\"Dai Stablecoin\",\"symbol\":\"DAI\",\"decimals\":18,\"weight\":50,\"swappable\":true}],\"extra\":\"{\\\"i\\\":\\\"100000\\\",\\\"K\\\":\\\"1000000000000000000\\\",\\\"B\\\":\\\"15580539464573\\\",\\\"Q\\\":\\\"54845488636364795\\\",\\\"B0\\\":\\\"2923221347601320894515\\\",\\\"Q0\\\":\\\"0\\\",\\\"R\\\":\\\"1\\\",\\\"mtFeeRate\\\":\\\"600000000000000\\\",\\\"lpFeeRate\\\":\\\"2400000000000000\\\",\\\"swappable\\\":true}\",\"staticExtra\":\"{\\\"poolId\\\":\\\"0x68276dc302d390245f3382eb4d2ea3a9317d46ef\\\",\\\"lpToken\\\":\\\"0x68276dc302d390245f3382eb4d2ea3a9317d46ef\\\",\\\"type\\\":\\\"DVM\\\",\\\"tokens\\\":[\\\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\\\",\\\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\\\"],\\\"dodoV1SellHelper\\\":\\\"0xa5f36e822540efd11fcd77ec46626b916b217c3e\\\"}\"}",
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
		// 0.1 WETH -> ? BERD
		{
			0,
			"0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
			"0x5330467941b3691a2c838769a58ddc5fca22ddec",
			bignumber.NewBig10("100000000000000000"),
			nil,
			nil,
			nil,
			libv2.ErrShouldNotBeZero,
		},

		// 0.01 WETH -> ? BERD
		{
			0,
			"0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
			"0x5330467941b3691a2c838769a58ddc5fca22ddec",
			bignumber.NewBig10("10000000000000000"),
			nil,
			nil,
			nil,
			libv2.ErrShouldNotBeZero,
		},

		// 1000 BERD -> ? WETH
		{
			0,
			"0x5330467941b3691a2c838769a58ddc5fca22ddec",
			"0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
			bignumber.NewBig10("1000000000000000000000"),
			nil,
			nil,
			nil,
			libv2.ErrTargetIsZero,
		},

		// 0.1 WETH -> ? DAI
		{
			1,
			"0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
			"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1",
			bignumber.NewBig10("100000000000000000"),
			bignumber.NewBig10("54672434201713829"),
			bignumber.NewBig10("0"),
			bignumber.NewBig10("164510835110472"),
			nil,
		},

		// 1 WETH -> ? DAI
		{
			1,
			"0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
			"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1",
			bignumber.NewBig10("1000000000000000000"),
			bignumber.NewBig10("54680100516436846"),
			bignumber.NewBig10("0"),
			bignumber.NewBig10("164533903259087"),
			nil,
		},

		// 100 DAI -> ? WETH
		{
			1,
			"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1",
			"0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
			bignumber.NewBig10("100000000000000000000"),
			bignumber.NewBig10("15525282928850"),
			bignumber.NewBig10("0"),
			bignumber.NewBig10("46715996776"),
			nil,
		},

		// 1000 DAI -> ? WETH
		{
			1,
			"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1",
			"0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
			bignumber.NewBig10("1000000000000000000000"),
			bignumber.NewBig10("15532945934166"),
			bignumber.NewBig10("0"),
			bignumber.NewBig10("46739054966"),
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
