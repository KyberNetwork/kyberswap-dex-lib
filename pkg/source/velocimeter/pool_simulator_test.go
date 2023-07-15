package velocimeter

import (
	"fmt"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalcAmountOut(t *testing.T) {
	// test data from https://ftmscan.com/address/0x0e8f117a563be78eb5a391a066d0d43dd187a9e0#readContract#F8
	testcases := []struct {
		in                string
		inAmount          string
		out               string
		expectedOutAmount string
	}{
		{"B", "100", "A", "40"},
		{"A", "100000000000", "B", "243206804406"},
	}

	p, err := NewPool(entity.Pool{
		Exchange:    "",
		Type:        "",
		SwapFee:     0.003, // from factory getFee https://ftmscan.com/address/0x472f3c3c9608fe0ae8d702f3f8a2d12c410c881a#readContract#F6
		Reserves:    entity.PoolReserves{"257894248517799332584152", "629103671583531892529021"},
		Tokens:      []*entity.PoolToken{{Address: "A", Decimals: 18}, {Address: "B", Decimals: 18}},
		StaticExtra: "{\"stable\": false}",
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
