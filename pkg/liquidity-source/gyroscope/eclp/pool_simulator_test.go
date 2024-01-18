package gyroeclp

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
)

func TestCalcAmountOut(t *testing.T) {
	t.Run("1. should return correct result", func(t *testing.T) {
		// input
		// polygon, block 51339771
		p := `{
			"address": "0x32fc95287b14eaef3afa92cccc48c285ee3a280a",
			"reserveUsd": 3454.483888331181,
			"amplifiedTvl": 3454.483888331181,
			"exchange": "balancer-v2-weighted",
			"type": "balancer-v2-weighted",
			"timestamp": 1703033832,
			"reserves": [
				"382259350067562080018",
				"563895201975090444069",
				"432276836",
				"415858931425966091248020",
				"198780894165507591",
				"9187067339281421763",
				"111172932376992452571",
				"1835599921140802978251"
			],
			"tokens": [
				{
					"address": "0x0b3f868e0be5597d5db7feb59e1cadbb0fdda50a",
					"name": "",
					"symbol": "",
					"decimals": 0,
					"weight": 0,
					"swappable": true
				},
				{
					"address": "0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270",
					"name": "",
					"symbol": "",
					"decimals": 0,
					"weight": 0,
					"swappable": true
				},
				{
					"address": "0x2791bca1f2de4661ed88a30c99a7a9449aa84174",
					"name": "",
					"symbol": "",
					"decimals": 0,
					"weight": 0,
					"swappable": true
				},
				{
					"address": "0x580a84c73811e1839f75d86d75d88cca0c241ff4",
					"name": "",
					"symbol": "",
					"decimals": 0,
					"weight": 0,
					"swappable": true
				},
				{
					"address": "0x7ceb23fd6bc0add59e62ac25578270cff1b9f619",
					"name": "",
					"symbol": "",
					"decimals": 0,
					"weight": 0,
					"swappable": true
				},
				{
					"address": "0x831753dd7087cac61ab5644b308642cc1c33dc13",
					"name": "",
					"symbol": "",
					"decimals": 0,
					"weight": 0,
					"swappable": true
				},
				{
					"address": "0x9a71012b13ca4d3d0cdc72a177df3ef03b0e76a3",
					"name": "",
					"symbol": "",
					"decimals": 0,
					"weight": 0,
					"swappable": true
				},
				{
					"address": "0xc3fdbadc7c795ef1d6ba111e06ff8f16a20ea539",
					"name": "",
					"symbol": "",
					"decimals": 0,
					"weight": 0,
					"swappable": true
				}
			],
			"extra": "{\"swapFeePercentage\":\"0x2386f26fc10000\",\"paused\":false}",
			"staticExtra": "{\"poolId\":\"0x32fc95287b14eaef3afa92cccc48c285ee3a280a000100000000000000000005\",\"poolType\":\"Weighted\",\"poolTypeVer\":1,\"scalingFactors\":[\"0x1\",\"0x1\",\"0xe8d4a51000\",\"0x1\",\"0x1\",\"0x1\",\"0x1\",\"0x1\"],\"normalizedWeights\":[\"0x1bc16d674ec8000\",\"0x1bc16d674ec8000\",\"0x1bc16d674ec8000\",\"0x1bc16d674ec8000\",\"0x1bc16d674ec8000\",\"0x1bc16d674ec8000\",\"0x1bc16d674ec8000\",\"0x1bc16d674ec8000\"],\"vault\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\"}"
		}`
		var pool entity.Pool
		err := json.Unmarshal([]byte(p), &pool)
		assert.Nil(t, err)

		// expected
		expectedAmountOut := "49523009318781117474536"

		// calculation
		simulator, err := NewPoolSimulator(pool)
		assert.Nil(t, err)
		amountIn, _ := new(big.Int).SetString("77000000000000000000", 10)
		result, err := simulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{
				Token:  "0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270",
				Amount: amountIn,
			},
			TokenOut: "0x580a84c73811e1839f75d86d75d88cca0c241ff4",
		})

		// assert
		assert.Nil(t, err)
		assert.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
	})

	t.Run("2. should return correct result", func(t *testing.T) {})

	t.Run("3. should return correct result", func(t *testing.T) {})

	t.Run("4. should return correct result", func(t *testing.T) {})

	t.Run("5. should return correct result", func(t *testing.T) {})
}
