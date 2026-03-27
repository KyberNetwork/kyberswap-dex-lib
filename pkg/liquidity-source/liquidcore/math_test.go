package liquidcore

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestCalcSwap(t *testing.T) {
	tests := []struct {
		name          string
		pool          *PoolState
		fromToken     string
		toToken       string
		amountIn      *uint256.Int
		wantAmountOut string
	}{
		{
			pool: &PoolState{
				Token0:      "token0",
				Token1:      "token1",
				Decimals0:   6,
				Decimals1:   18,
				Reserve0:    uint256.NewInt(17802215630),
				Reserve1:    uint256.MustFromDecimal("231660699390171237626"),
				SpotPrice:   uint256.NewInt(38578000),
				OraclePrice: uint256.NewInt(385740),
			},
			fromToken:     "token1",
			toToken:       "token0",
			amountIn:      uint256.MustFromDecimal("100000000000000000000"),
			wantAmountOut: "3856911385",
		},
		{
			pool: &PoolState{
				Token0:      "token0",
				Token1:      "token1",
				Decimals0:   6,
				Decimals1:   18,
				Reserve0:    uint256.NewInt(17814953089),
				Reserve1:    uint256.MustFromDecimal("251762699390171237626"),
				SpotPrice:   uint256.NewInt(38545000),
				OraclePrice: uint256.NewInt(385360),
			},
			fromToken:     "token1",
			toToken:       "token0",
			amountIn:      uint256.MustFromDecimal("20102000000000000000"),
			wantAmountOut: "774658848",
		},
		{
			pool: &PoolState{
				Token0:      "token0",
				Token1:      "token1",
				Decimals0:   6,
				Decimals1:   18,
				Reserve0:    uint256.NewInt(16174442279),
				Reserve1:    uint256.MustFromDecimal("273571523887411554367"),
				SpotPrice:   uint256.NewInt(39197000),
				OraclePrice: uint256.NewInt(391980),
			},
			fromToken:     "token1",
			toToken:       "token0",
			amountIn:      uint256.MustFromDecimal("20102000000000000000"),
			wantAmountOut: "787745943",
		},
		{
			pool: &PoolState{
				Token0:      "token0",
				Token1:      "token1",
				Decimals0:   6,
				Decimals1:   18,
				Reserve0:    uint256.NewInt(16554601967),
				Reserve1:    uint256.MustFromDecimal("263824054468989456062"),
				SpotPrice:   uint256.NewInt(38935000),
				OraclePrice: uint256.NewInt(389320),
			},
			fromToken:     "token1",
			toToken:       "token0",
			amountIn:      uint256.MustFromDecimal("201020000000000"),
			wantAmountOut: "7825",
		},
		{
			pool: &PoolState{
				Token0:      "token0",
				Token1:      "token1",
				Decimals0:   6,
				Decimals1:   18,
				Reserve0:    uint256.NewInt(11558777983),
				Reserve1:    uint256.MustFromDecimal("221962947936423726550"),
				SpotPrice:   uint256.NewInt(38898000),
				OraclePrice: uint256.NewInt(388900),
			},
			fromToken:     "token1",
			toToken:       "token0",
			amountIn:      uint256.MustFromDecimal("1000000000000000000"),
			wantAmountOut: "38888673",
		},
		{
			pool: &PoolState{
				Token0:      "token0",
				Token1:      "token1",
				Decimals0:   6,
				Decimals1:   18,
				Reserve0:    uint256.NewInt(16599055255),
				Reserve1:    uint256.MustFromDecimal("251245153037908374635"),
				SpotPrice:   uint256.NewInt(38792000),
				OraclePrice: uint256.NewInt(387970),
			},
			fromToken:     "token1",
			toToken:       "token0",
			amountIn:      uint256.MustFromDecimal("201020000000000"),
			wantAmountOut: "7797",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CalcSwap(tt.pool, tt.fromToken, tt.toToken, tt.amountIn)
			require.NoError(t, err)
			require.Equal(t, tt.wantAmountOut, result.AmountOut.Dec())
		})
	}
}
