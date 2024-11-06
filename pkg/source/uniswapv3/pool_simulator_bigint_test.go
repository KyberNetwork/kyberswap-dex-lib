package uniswapv3

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestPoolSimulatorBigInt_CalcAmountIn(t *testing.T) {
	type testcase struct {
		name        string
		poolEncoded string
		tokenOut    string
		amountOut   string
		tokenIn     string
	}
	testcases := []testcase{
		{
			name: "swap WETH for USDT",
			poolEncoded: `{
				"address": "0x43c3ca9dc59f5144a18b3a4fd017c692bda05026",
				"swapFee": 3000,
				"exchange": "blueprint",
				"type": "uniswapv3",
				"timestamp": 1711596217,
				"reserves": [
					"308271459832077103",
					"1245211697"
				],
				"tokens": [
					{
						"address": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
						"name": "Wrapped Ether",
						"symbol": "WETH",
						"decimals": 18,
						"weight": 0,
						"swappable": true
					},
					{
						"address": "0xdac17f958d2ee523a2206206994597c13d831ec7",
						"name": "USDT",
						"symbol": "USDT",
						"decimals": 6,
						"weight": 0,
						"swappable": true
					}
				],
				"extra": "{\"liquidity\":0,\"sqrtPriceX96\":4736796043499064985766337,\"tick\":-194505,\"ticks\":[{\"index\":-197760,\"liquidityGross\":5158244147190,\"liquidityNet\":5158244147190},{\"index\":-196320,\"liquidityGross\":5158244147190,\"liquidityNet\":-5158244147190}]}",	
				"staticExtra": "{\"poolId\":\"0x43c3ca9dc59f5144a18b3a4fd017c692bda05026\"}"
			}`,
			tokenOut:  "0xdac17f958d2ee523a2206206994597c13d831ec7",
			amountOut: "500000000", // 500 USDT
			tokenIn:   "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			poolEntity := new(entity.Pool)
			err := json.Unmarshal([]byte(tc.poolEncoded), poolEntity)
			require.NoError(t, err)

			poolSim, err := NewPoolSimulatorBigInt(*poolEntity, valueobject.ChainIDEthereum)
			require.NoError(t, err)

			result, err := testutil.MustConcurrentSafe[*pool.CalcAmountInResult](t, func() (any, error) {
				return poolSim.CalcAmountIn(pool.CalcAmountInParams{
					TokenAmountOut: pool.TokenAmount{
						Token:  tc.tokenOut,
						Amount: bignumber.NewBig10(tc.amountOut),
					},
					TokenIn: tc.tokenIn,
				})
			})
			require.NoError(t, err)
			assert.Equal(t, big.NewInt(7074025631378098), result.TokenAmountIn.Amount)
			assert.Equal(t, big.NewInt(-480436293), result.RemainingTokenAmountOut.Amount)
		})
	}
}
