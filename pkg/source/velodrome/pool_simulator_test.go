package velodrome

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCalcAmountOut(t *testing.T) {
	// test data from https://optimistic.etherscan.io/address/0x79c912fef520be002c2b6e57ec4324e260f38e50#readContract
	testcases := []struct {
		in                string
		inAmount          string
		out               string
		expectedOutAmount string
		reserveA          string
		reserveB          string
		swapFee           float64
		expectError       bool
	}{
		{"B", "100", "A", "172023517829", "2082415614000308399878", "3631620514949", 0.0005, false},
		{"A", "100000000000", "B", "58", "2082415614000308399878", "3631620514949", 0.0005, false},
		{"A", "1631", "B", "0", "503877670", "1", 0.0005, true}, // do not allow swapping the entire reserve in the pool
	}

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			p, err := NewPoolSimulator(entity.Pool{
				Exchange:    "",
				Type:        "",
				SwapFee:     tc.swapFee,
				Reserves:    entity.PoolReserves{tc.reserveA, tc.reserveB},
				Tokens:      []*entity.PoolToken{{Address: "A", Decimals: 18}, {Address: "B", Decimals: 6}},
				StaticExtra: "{\"stable\": true}",
			})
			require.Nil(t, err)

			assert.Equal(t, []string{"A"}, p.CanSwapTo("B"))
			assert.Equal(t, []string{"B"}, p.CanSwapTo("A"))

			amountIn := pool.TokenAmount{Token: tc.in, Amount: bignumber.NewBig10(tc.inAmount)}
			out, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return p.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: amountIn,
					TokenOut:      tc.out,
				})
			})

			if tc.expectError {
				assert.Error(t, err, "Expected an error but got nil")
			} else {
				require.NoError(t, err, "Did not expect an error but got one")
				assert.Equal(t, bignumber.NewBig10(tc.expectedOutAmount), out.TokenAmountOut.Amount)
				assert.Equal(t, tc.out, out.TokenAmountOut.Token)
			}
		})
	}
}
