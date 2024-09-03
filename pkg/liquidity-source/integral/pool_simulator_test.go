package integral

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestCalcAmountOut(t *testing.T) {
	t.Run("1. should return OK", func(t *testing.T) {

		extraBytes, err := json.Marshal(IntegralPair{
			SwapFee:           uint256.NewInt(10 ^ 14),
			DecimalsConverter: big.NewInt(1000000),
			AveragePrice:      uint256.NewInt(399733926911723),
		})
		require.Nil(t, err)

		var pool = entity.Pool{
			Address: "",
			SwapFee: 0.0001,
			Reserves: entity.PoolReserves{
				"258594532323",
				"210054008990797983557",
			},
			Tokens: []*entity.PoolToken{
				{Address: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"},
				{Address: "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"},
			},
			Extra: string(extraBytes),
		}

		// expected amount
		expectedAmountOut := "3966282853083561"

		// calculation
		simulator, err := NewPoolSimulator(pool)
		require.Nil(t, err)

		amountIn, _ := new(big.Int).SetString("10000000", 10)
		result, err := testutil.MustConcurrentSafe[*poolpkg.CalcAmountOutResult](t, func() (any, error) {
			return simulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Token:  "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
					Amount: amountIn,
				},
				TokenOut: "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2",
			})
		})

		// assert
		require.Nil(t, err)
		require.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
	})
}
