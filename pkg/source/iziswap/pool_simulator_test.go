package iziswap

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestCalcAmountOut(t *testing.T) {
	// test data from https://polygonscan.com/address/0xee45cffbfafe97691b8ef068c8d55163086a3431
	testcases := []struct {
		in                string
		inAmount          string
		out               string
		expectedOutAmount string
	}{
		{"A", "2000000000000000000", "B", "18037620383221447462"},
		{"B", "2000000000000000000", "A", "110876282573252914"},
	}

	p, err := NewPoolSimulator(entity.Pool{
		Address:  "0xee45cffbfafe97691b8ef068c8d55163086a3431",
		Exchange: "iziswap",
		Type:     "iziswap",
		SwapFee:  400,
		Reserves: entity.PoolReserves{"1167087113545385273", "18037620383221447465"},
		Tokens:   []*entity.PoolToken{{Address: "A", Decimals: 18}, {Address: "B", Decimals: 18}},
		Extra:    "{\"CurrentPoint\":28912,\"PointDelta\":8,\"LeftMostPt\":-800000,\"RightMostPt\":800000,\"Fee\":400,\"Liquidity\":23123688144702854,\"LiquidityX\":8210612878032008,\"Liquidities\":[{\"LiqudityDelta\":23123688144702854,\"Point\":28728},{\"LiqudityDelta\":-23123688144702854,\"Point\":29128}],\"LimitOrders\":[]}",
	})
	require.Nil(t, err)

	assert.Equal(t, []string{"A"}, p.CanSwapTo("B"))
	assert.Equal(t, []string{"B"}, p.CanSwapTo("A"))

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			amountIn := pool.TokenAmount{Token: tc.in, Amount: bignumber.NewBig10(tc.inAmount)}
			out, err := p.CalcAmountOut(amountIn, tc.out)
			require.Nil(t, err)
			assert.Equal(t, bignumber.NewBig10(tc.expectedOutAmount), out.TokenAmountOut.Amount)
			assert.Equal(t, tc.out, out.TokenAmountOut.Token)
		})
	}
}
