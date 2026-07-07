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
		// Existing 6/18 pools (USDC/HYPE-type): FP*IP ≈ 1e12, scale = 1e6
		{
			pool: &PoolState{
				Token0:    "token0",
				Decimals0: 6,
				Decimals1: 18,
				Reserve0:  uint256.NewInt(17802215630),
				Reserve1:  uint256.MustFromDecimal("231660699390171237626"),
				SpotPrice: uint256.NewInt(38578000),
			},
			fromToken:     "token1",
			toToken:       "token0",
			amountIn:      uint256.MustFromDecimal("100000000000000000000"),
			wantAmountOut: "3856835550",
		},
		{
			pool: &PoolState{
				Token0:    "token0",
				Decimals0: 6,
				Decimals1: 18,
				Reserve0:  uint256.NewInt(17814953089),
				Reserve1:  uint256.MustFromDecimal("251762699390171237626"),
				SpotPrice: uint256.NewInt(38545000),
			},
			fromToken:     "token1",
			toToken:       "token0",
			amountIn:      uint256.MustFromDecimal("20102000000000000000"),
			wantAmountOut: "774637883",
		},
		{
			pool: &PoolState{
				Token0:    "token0",
				Decimals0: 6,
				Decimals1: 18,
				Reserve0:  uint256.NewInt(16174442279),
				Reserve1:  uint256.MustFromDecimal("273571523887411554367"),
				SpotPrice: uint256.NewInt(39197000),
			},
			fromToken:     "token1",
			toToken:       "token0",
			amountIn:      uint256.MustFromDecimal("20102000000000000000"),
			wantAmountOut: "787741110",
		},
		{
			pool: &PoolState{
				Token0:    "token0",
				Decimals0: 6,
				Decimals1: 18,
				Reserve0:  uint256.NewInt(16554601967),
				Reserve1:  uint256.MustFromDecimal("263824054468989456062"),
				SpotPrice: uint256.NewInt(38935000),
			},
			fromToken:     "token1",
			toToken:       "token0",
			amountIn:      uint256.MustFromDecimal("201020000000000"),
			wantAmountOut: "7825",
		},
		{
			pool: &PoolState{
				Token0:    "token0",
				Decimals0: 6,
				Decimals1: 18,
				Reserve0:  uint256.NewInt(11558777983),
				Reserve1:  uint256.MustFromDecimal("221962947936423726550"),
				SpotPrice: uint256.NewInt(38898000),
			},
			fromToken:     "token1",
			toToken:       "token0",
			amountIn:      uint256.MustFromDecimal("1000000000000000000"),
			wantAmountOut: "38888276",
		},
		{
			pool: &PoolState{
				Token0:    "token0",
				Decimals0: 6,
				Decimals1: 18,
				Reserve0:  uint256.NewInt(16599055255),
				Reserve1:  uint256.MustFromDecimal("251245153037908374635"),
				SpotPrice: uint256.NewInt(38792000),
			},
			fromToken:     "token1",
			toToken:       "token0",
			amountIn:      uint256.MustFromDecimal("201020000000000"),
			wantAmountOut: "7796",
		},
		// UBTC(8)/UETH(18) pool: FP*IP ≈ 1e20, scale = 1e10
		// On-chain: 0x437bccdb2875aace0f685fc7e730b0a758346e5e @ block 36634918
		// estimateSwap(UBTC, UETH, 1000000) on-chain = 367393297419180338 (includes fee)
		{
			name: "UBTC/UETH 8-dec/18-dec pool",
			pool: &PoolState{
				Token0:    "0x9fdbda0a5e284c32744d2f17ee5c74b284993463",
				Decimals0: 8,
				Decimals1: 18,
				Reserve0:  uint256.NewInt(79582613),
				Reserve1:  uint256.MustFromDecimal("8203190593497158815"),
				SpotPrice: uint256.NewInt(272151553),
			},
			fromToken:     "0x9fdbda0a5e284c32744d2f17ee5c74b284993463",
			toToken:       "0xbe6727b535545c67d5caa73dea54865b92cf7907",
			amountIn:      uint256.NewInt(1_000_000),
			wantAmountOut: "365186231364257547",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CalcSwap(tt.pool, tt.fromToken, tt.amountIn)
			require.NoError(t, err)
			require.Equal(t, tt.wantAmountOut, result.AmountOut.Dec())
		})
	}
}
