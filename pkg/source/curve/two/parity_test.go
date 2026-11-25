package two

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// goldenRow is one quote captured from the post-fix baseline oracle (see testdata/README note in the commit body).
type goldenRow struct {
	Pool, In, Out, Amount, Dy, Fee, Err string
}

// negativeResultRows are the golden rows whose native quote underflows where the oracle returned a value-less
// tokenIndex error. Each is a quote whose output would be zero or negative; they must fail natively.
var negativeResultRows = map[string]bool{
	"cvxETH 0x4e3f->0xc02a amt=1":    true,
	"cvxETH 0x4e3f->0xc02a amt=2":    true,
	"cvxETH 0x4e3f->0xc02a amt=10":   true,
	"cvxETH 0x4e3f->0xc02a amt=1000": true,
	"tETH 0xc02a->0xcdf7 amt=1":      true,
	"tETH 0xc02a->0xcdf7 amt=2":      true,
	"tETH 0xc02a->0xcdf7 amt=10":     true,
	"tETH 0xcdf7->0xc02a amt=1":      true,
	"tETH 0xcdf7->0xc02a amt=2":      true,
	"tETH 0xcdf7->0xc02a amt=10":     true,
	"tETH 0xcdf7->0xc02a amt=1000":   true,
	"eursUSDC 0xa0b8->0xdb25 amt=0":  true,
	"eursUSDC 0xdb25->0xa0b8 amt=0":  true,
}

func loadGolden(t *testing.T) []goldenRow {
	t.Helper()
	b, err := os.ReadFile("testdata/legacy_goldens.json")
	require.NoError(t, err)
	var rows []goldenRow
	require.NoError(t, json.Unmarshal(b, &rows))
	return rows
}

// TestLegacyParity replays every golden quote through the native uint256 CalcAmtOut. Every row must match the
// post-fix baseline exactly, except the negative-result rows, which must fail natively.
func TestLegacyParity(t *testing.T) {
	t.Parallel()
	rows := loadGolden(t)
	require.Len(t, rows, 24000)

	pools := map[string]*PoolSimulator{}
	var checkedNegative int
	for idx, c := range rows {
		p := pools[c.Pool]
		if p == nil {
			var err error
			p, err = NewPoolSimulator(allStates[c.Pool].entityPool())
			require.NoError(t, err)
			pools[c.Pool] = p
		}
		key := fmt.Sprintf("%s %s->%s amt=%s", c.Pool, c.In[:6], c.Out[:6], c.Amount)
		amt, ok := new(big.Int).SetString(c.Amount, 10)
		require.True(t, ok, "row %d amount", idx)
		res, err := p.CalcAmtOut(pool.CalcAmtOutParams{
			TokenAmountIn: pool.TokenAmt{Token: c.In, Amount: mustU256(t, amt)},
			TokenOut:      c.Out,
		})
		if negativeResultRows[key] {
			require.Error(t, err, "row %d %s: native must reject the negative result", idx, key)
			require.Contains(t, err.Error(), "underflow", "row %d %s", idx, key)
			checkedNegative++
			continue
		}
		if c.Err != "" {
			require.Error(t, err, "row %d %s: want legacy error %q", idx, key, c.Err)
			require.Equal(t, c.Err, err.Error(), "row %d %s", idx, key)
			continue
		}
		require.NoError(t, err, "row %d %s", idx, key)
		require.Equal(t, c.Dy, res.TokenAmountOut.Amount.String(), "row %d %s: amount out", idx, key)
	}
	require.Equal(t, 15, checkedNegative, "every negative-result row must be checked")
}

func mustU256(t *testing.T, b *big.Int) (z uint256.Int) {
	t.Helper()
	v, err := toU256(b)
	require.NoError(t, err)
	return v
}
