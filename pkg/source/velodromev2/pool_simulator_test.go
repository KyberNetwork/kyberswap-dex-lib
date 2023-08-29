package velodromev2

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
	// test data from https://optimistic.etherscan.io/address/0x79c912fef520be002c2b6e57ec4324e260f38e50#readContract
	testcases := []struct {
		in                string
		inAmount          string
		out               string
		expectedOutAmount string
	}{
		{"B", "100", "A", "57341222888"},
		{"A", "100000000000", "B", "174"},
	}

	p, err := NewPoolSimulator(entity.Pool{
		Exchange:    "",
		Type:        "",
		SwapFee:     0.0005, // from factory https://optimistic.etherscan.io/address/0x25cbddb98b35ab1ff77413456b31ec81a6b6b746#readContract
		Reserves:    entity.PoolReserves{"2082415614000308399878", "3631620514949"},
		Tokens:      []*entity.PoolToken{{Address: "A", Decimals: 18}, {Address: "B", Decimals: 6}},
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
