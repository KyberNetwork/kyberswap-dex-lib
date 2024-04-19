package ezeth

import (
	"testing"

	"github.com/stretchr/testify/assert"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	// https://etherscan.io/tx/0xe78708a81ecb2cc6ac6906e251d1dc5fd10bec32e421250b88f9c5127d2074f9
	t.Run("[DepositETH] it should return correct amountOut", func(t *testing.T) {
		poolSimulator := PoolSimulator{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Tokens: []string{
						"0xbf5495efe5db9ce00f80364c8b423567e58d2110",
						"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
						"0xa2e3356610840701bdf5611a53974510ae27e2e1",
						"0xae7ab96520de3a18e5e111b5eaab095312d7fe84",
					},
				},
			},
			paused:        false,
			totalTVL:      bignumber.NewBig("846148216510217972629804"),
			totalSupply:   bignumber.NewBig("839310921147858962585526"),
			maxDepositTVL: bignumber.ZeroBI,
		}

		params := poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{
				Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				Amount: bignumber.NewBig("1000000000000000000"),
			},
			TokenOut: "0xbf5495efe5db9ce00f80364c8b423567e58d2110",
		}

		result, err := poolSimulator.CalcAmountOut(params)

		assert.NoError(t, err)
		assert.Equal(t, bignumber.NewBig("991919506265011106"), result.TokenAmountOut.Amount)
	})

	// https://etherscan.io/tx/0x95dce3940863dda4c7506a2080b7edbfdc26769641a75730f2b2e6bfa7a33d06
	t.Run("[Deposit] it should return correct amountOut", func(t *testing.T) {
	})
}
