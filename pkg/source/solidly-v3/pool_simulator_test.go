package solidlyv3

import (
	"math/big"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestCalcAmountOutConcurrentSafe(t *testing.T) {
	type testcase struct {
		name        string
		poolEncoded string
		tokenIn     string
		amountIn    string
		tokenOut    string
	}
	testcases := []testcase{
		{
			name: "swap USDT for USDC",
			poolEncoded: `{
				"address": "0x6146be494fee4c73540cb1c5f87536abf1452500",
				"swapFee": 100,
				"type": "solidly-v3",
				"timestamp": 1705358961,
				"reserves": [
					"137746578201",
					"1484208757880"
				],
				"tokens": [
					{
						"address": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
						"name": "USD Coin",
						"symbol": "USDC",
						"decimals": 6,
						"weight": 50,
						"swappable": true
					},
					{
						"address": "0xdac17f958d2ee523a2206206994597c13d831ec7",
						"name": "Tether USD",
						"symbol": "USDT",
						"decimals": 6,
						"weight": 50,
						"swappable": true
					}
				],
				"extra": "{\"liquidity\":2187336922123374,\"sqrtPriceX96\":79257523413855207489606556516,\"tickSpacing\":1,\"tick\":7,\"ticks\":[{\"index\":-51,\"liquidityGross\":4023735441,\"liquidityNet\":4023735441},{\"index\":-20,\"liquidityGross\":5377610146132,\"liquidityNet\":5377610146132},{\"index\":-15,\"liquidityGross\":210582075003910,\"liquidityNet\":210582075003910},{\"index\":-12,\"liquidityGross\":22472177151754,\"liquidityNet\":22472177151754},{\"index\":-10,\"liquidityGross\":74074635783340,\"liquidityNet\":74074635783340},{\"index\":-8,\"liquidityGross\":433140695370165,\"liquidityNet\":433140695370165},{\"index\":-7,\"liquidityGross\":39366634237420,\"liquidityNet\":39366634237420},{\"index\":-6,\"liquidityGross\":500,\"liquidityNet\":500},{\"index\":-5,\"liquidityGross\":426569798759873,\"liquidityNet\":372153403538201},{\"index\":-3,\"liquidityGross\":1219258653378841,\"liquidityNet\":1219258653378841},{\"index\":-2,\"liquidityGross\":22472177151754,\"liquidityNet\":-22472177151754},{\"index\":-1,\"liquidityGross\":895763774632318,\"liquidityNet\":740462860612512},{\"index\":0,\"liquidityGross\":1110632836915778,\"liquidityNet\":-1110632836915778},{\"index\":1,\"liquidityGross\":87397254001105,\"liquidityNet\":19516406989539},{\"index\":3,\"liquidityGross\":191372132878096,\"liquidityNet\":177871973029250},{\"index\":5,\"liquidityGross\":1059772830149054,\"liquidityNet\":406901047912696},{\"index\":6,\"liquidityGross\":997092488198795,\"liquidityNet\":-997092488198795},{\"index\":7,\"liquidityGross\":596352227500000,\"liquidityNet\":596352227500000},{\"index\":8,\"liquidityGross\":1139269436790704,\"liquidityNet\":-1139269436790704},{\"index\":9,\"liquidityGross\":649809057995322,\"liquidityNet\":-649809057995322},{\"index\":10,\"liquidityGross\":398254403601907,\"liquidityNet\":-398254403601907},{\"index\":50,\"liquidityGross\":4023735441,\"liquidityNet\":-4023735441}]}"
			}`,
			tokenIn:  "0xdac17f958d2ee523a2206206994597c13d831ec7",
			amountIn: "1000000000", // 1000
			tokenOut: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			poolEntity := new(entity.Pool)
			err := sonic.Unmarshal([]byte(tc.poolEncoded), poolEntity)
			require.NoError(t, err)

			poolSim, err := NewPoolSimulator(*poolEntity, valueobject.ChainIDEthereum)
			require.NoError(t, err)

			result, err := testutil.MustConcurrentSafe[*pool.CalcAmountOutResult](t, func() (any, error) {
				return poolSim.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{
						Token:  tc.tokenIn,
						Amount: bignumber.NewBig10(tc.amountIn),
					},
					TokenOut: tc.tokenOut,
				})
			})
			require.NoError(t, err)
			_ = result
		})
	}
}

func TestPoolSimulator_CalcAmountIn(t *testing.T) {
	type testcase struct {
		name        string
		poolEncoded string
		tokenOut    string
		amountOut   string
		tokenIn     string
	}
	testcases := []testcase{
		{
			name: "swap WETH for AI",
			poolEncoded: `{
				"address":"0xfc9e7373109adacd18152cc24658bf8b34ac3dba","reserveUsd":516.1427089129024,"amplifiedTvl":4.126356361103288e+47,"swapFee":10000,"exchange":"solidly-v3","type":"solidly-v3","timestamp":1710154644,"reserves":["6897657865010157199229186","34285160896988154"],"tokens":[{"address":"0x2598c30330d5771ae9f983979209486ae26de875","name":"Any Inu","symbol":"AI","decimals":18,"weight":50,"swappable":true},{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","name":"Wrapped Ether","symbol":"WETH","decimals":18,"weight":50,"swappable":true}],"extra":"{\"liquidity\":11420043566174051417,\"sqrtPriceX96\":9374274824798812391640411,\"tickSpacing\":100,\"tick\":-180852,\"ticks\":[{\"index\":-887200,\"liquidityGross\":11420043566174051417,\"liquidityNet\":11420043566174051417},{\"index\":-191100,\"liquidityGross\":84172845905035329535,\"liquidityNet\":84172845905035329535},{\"index\":-185000,\"liquidityGross\":84172845905035329535,\"liquidityNet\":-84172845905035329535},{\"index\":887200,\"liquidityGross\":11420043566174051417,\"liquidityNet\":-11420043566174051417}]}"
			}`,
			tokenOut:  "0x2598c30330d5771ae9f983979209486ae26de875",
			amountOut: "10000000000000000000000", // 10000 AI
			tokenIn:   "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			poolEntity := new(entity.Pool)
			err := sonic.Unmarshal([]byte(tc.poolEncoded), poolEntity)
			require.NoError(t, err)

			poolSim, err := NewPoolSimulator(*poolEntity, valueobject.ChainIDEthereum)
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
			assert.Equal(t, big.NewInt(157754838261356), result.TokenAmountIn.Amount)
			assert.Equal(t, big.NewInt(0), result.RemainingTokenAmountOut.Amount)
		})
	}
}
