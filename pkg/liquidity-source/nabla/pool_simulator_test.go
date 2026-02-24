package nabla

import (
	"testing"

	"github.com/KyberNetwork/int256"
	"github.com/stretchr/testify/require"
)

func Test_sell(t *testing.T) {
	p0Meta := NablaPoolMeta{
		CurveBeta:   int256.NewInt(5000000000000000),
		CurveC:      int256.MustFromDec("17075887234393789126"),
		BackstopFee: int256.NewInt(300),
		ProtocolFee: int256.NewInt(100),
		LpFee:       int256.NewInt(200),
	}

	p1Meta := NablaPoolMeta{
		CurveBeta:   int256.NewInt(5000000000000000),
		CurveC:      int256.MustFromDec("17075887234393789126"),
		BackstopFee: int256.NewInt(300),
		ProtocolFee: int256.NewInt(100),
		LpFee:       int256.NewInt(200),
	}

	p0StateBalanced := NablaPoolState{
		Reserve:             int256.MustFromDec("1000000000000000000000"),
		ReserveWithSlippage: int256.MustFromDec("1000000000000000000000"),
		TotalLiabilities:    int256.MustFromDec("1000000000000000000000"),
		Price:               int256.MustFromDec("100000000"),
	}

	p1StateBalanced := NablaPoolState{
		Reserve:             int256.MustFromDec("1000000000000000000000"),
		ReserveWithSlippage: int256.MustFromDec("1000000000000000000000"),
		TotalLiabilities:    int256.MustFromDec("1000000000000000000000"),
		Price:               int256.MustFromDec("100000000"),
	}

	p0StateImbalanced := NablaPoolState{
		Reserve:             int256.MustFromDec("1099997249253573525485"),
		ReserveWithSlippage: int256.MustFromDec("1100000000000000000000"),
		TotalLiabilities:    int256.MustFromDec("1000000000000000000000"),
		Price:               int256.MustFromDec("100000000"),
	}

	p1StateImbalanced := NablaPoolState{
		Reserve:             int256.MustFromDec("900052749371053261277"),
		ReserveWithSlippage: int256.MustFromDec("900055528992381704580"),
		TotalLiabilities:    int256.MustFromDec("1000019999449850714705"),
		Price:               int256.MustFromDec("100000000"),
	}

	tests := []struct {
		name     string
		p0State  NablaPoolState
		p1State  NablaPoolState
		fromIdx  int
		toIdx    int
		amountIn *int256.Int
		want     *int256.Int
	}{
		{
			name:     "swap with balanced base pools - p0 to p1",
			p0State:  p0StateBalanced,
			p1State:  p1StateBalanced,
			fromIdx:  0,
			toIdx:    1,
			amountIn: int256.MustFromDec("1000000000000000000"),
			want:     int256.MustFromDec("999399447164390453"),
		},
		{
			name:     "swap with balanced base pools - p1 to p0",
			p0State:  p0StateBalanced,
			p1State:  p1StateBalanced,
			fromIdx:  1,
			toIdx:    0,
			amountIn: int256.MustFromDec("1000000000000000000"),
			want:     int256.MustFromDec("999399447164390453"),
		},
		{
			name:     "swap with imbalanced pools - p0 to p1",
			p0State:  p0StateImbalanced,
			p1State:  p1StateImbalanced,
			fromIdx:  0,
			toIdx:    1,
			amountIn: int256.MustFromDec("1000000000000000000"),
			want:     int256.MustFromDec("999288878655122808"),
		},
		{
			name:     "swap with imbalanced pools - p1 to p0",
			p0State:  p0StateImbalanced,
			p1State:  p1StateImbalanced,
			fromIdx:  1,
			toIdx:    0,
			amountIn: int256.MustFromDec("100000000000000000000"),
			want:     int256.MustFromDec("99945528699304486120"),
		},
		{
			name: "swap with imbalanced pools - p0 to p1 with price update (case 1)",
			p0State: NablaPoolState{
				Reserve:             int256.MustFromDec("1099997249253573525485"),
				ReserveWithSlippage: int256.MustFromDec("1100000000000000000000"),
				TotalLiabilities:    int256.MustFromDec("1000000000000000000000"),
				Price:               int256.MustFromDec("200000000"),
			},
			p1State:  p1StateImbalanced,
			fromIdx:  0,
			toIdx:    1,
			amountIn: int256.MustFromDec("1000000000000000000"),
			want:     int256.MustFromDec("1998577195258299942"),
		},
		{
			name: "swap with imbalanced pools - p0 to p1 with price update (case 2)",
			p0State: NablaPoolState{
				Reserve:             int256.MustFromDec("1099997249253573525485"),
				ReserveWithSlippage: int256.MustFromDec("1100000000000000000000"),
				TotalLiabilities:    int256.MustFromDec("1000000000000000000000"),
				Price:               int256.MustFromDec("200000000"),
			},
			p1State:  p1StateImbalanced,
			fromIdx:  0,
			toIdx:    1,
			amountIn: int256.MustFromDec("294110300000000000000"),
			want:     int256.MustFromDec("587655876076133933446"),
		},
		{
			name: "swap with imbalanced pools - p1 to p0 with price update (case 1)",
			p0State: NablaPoolState{
				Reserve:             int256.MustFromDec("1099997249253573525485"),
				ReserveWithSlippage: int256.MustFromDec("1100000000000000000000"),
				TotalLiabilities:    int256.MustFromDec("1000000000000000000000"),
				Price:               int256.MustFromDec("200000000"),
			},
			p1State:  p1StateImbalanced,
			fromIdx:  1,
			toIdx:    0,
			amountIn: int256.MustFromDec("100000000000000000000"),
			want:     int256.MustFromDec("49973449671535241267"),
		},
		{
			name: "swap with imbalanced pools - p1 to p0 with price update (case 2)",
			p0State: NablaPoolState{
				Reserve:             int256.MustFromDec("1099997249253573525485"),
				ReserveWithSlippage: int256.MustFromDec("1100000000000000000000"),
				TotalLiabilities:    int256.MustFromDec("1000000000000000000000"),
				Price:               int256.MustFromDec("200000000"),
			},
			p1State:  p1StateImbalanced,
			fromIdx:  1,
			toIdx:    0,
			amountIn: int256.MustFromDec("73248000000000000000"),
			want:     int256.MustFromDec("36604959022922925425"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p0 := NablaPool{
				Meta:  p0Meta,
				State: tt.p0State,
			}
			p1 := NablaPool{
				Meta:  p1Meta,
				State: tt.p1State,
			}

			var fr, to NablaPool
			if tt.fromIdx == 0 {
				fr = p0
				to = p1
			} else {
				fr = p1
				to = p0
			}

			got, _, err := sell(fr, to, tt.amountIn, 18, 18)
			require.NoError(t, err)
			require.Equal(t, tt.want.Dec(), got.Dec())
		})
	}
}
