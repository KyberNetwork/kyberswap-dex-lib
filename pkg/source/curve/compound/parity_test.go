package compound

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// legacyGolden is the unmodified big.Int simulator's output, captured before the uint256 port.
type legacyGolden struct {
	Pool   string
	Quotes []goldenRow
	Ins    []goldenRow
	Upd    []string
}

type goldenRow struct {
	In, Out, Amount, Dy, Fee, Err string
}

// changedRows lists the golden rows whose output changes under the negative-result rule (decision D /
// B-prime). Every other row must match legacy exactly.
var changedRows = map[string]string{
	"Quotes[14]": "negative result",
	"Quotes[15]": "negative result",
	"Quotes[16]": "negative result",
	"Quotes[17]": "negative result",
	"Quotes[18]": "negative result",
	"Quotes[19]": "negative result",
	"Quotes[20]": "negative result",
	"Quotes[21]": "negative result",
	"Ins[7]":     "negative result",
	"Ins[8]":     "negative result",
	"Ins[9]":     "negative result",
	"Ins[10]":    "negative result",
	"Ins[21]":    "negative result",
}

func quoteParity[T any](f func() (T, error)) (res T, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return f()
}

func loadCompoundGolden(t *testing.T) legacyGolden {
	b, err := os.ReadFile("testdata/legacy_goldens.json")
	require.NoError(t, err)
	var g legacyGolden
	require.NoError(t, json.Unmarshal(b, &g))
	return g
}

// TestLegacyParity replays the compound golden through the native uint256 surface.
func TestLegacyParity(t *testing.T) {
	t.Parallel()
	g := loadCompoundGolden(t)
	var e entity.Pool
	require.NoError(t, json.Unmarshal([]byte(g.Pool), &e))

	for idx, c := range g.Quotes {
		key := fmt.Sprintf("Quotes[%d]", idx)
		p, err := NewPoolSimulator(e)
		require.NoError(t, err)
		amt := mustCU(t, c.Amount)
		res, err := quoteParity(func() (*pool.CalcAmtOutResult, error) {
			return p.CalcAmtOut(pool.CalcAmtOutParams{TokenAmountIn: pool.TokenAmt{Token: c.In, Amount: *amt}, TokenOut: c.Out})
		})
		checkCompoundRow(t, key, c, err, res, nil)
	}
	for idx, c := range g.Ins {
		key := fmt.Sprintf("Ins[%d]", idx)
		p, err := NewPoolSimulator(e)
		require.NoError(t, err)
		amt := mustCU(t, c.Amount)
		res, err := quoteParity(func() (*pool.CalcAmtInResult, error) {
			return p.CalcAmtIn(pool.CalcAmtInParams{TokenAmountOut: pool.TokenAmt{Token: c.Out, Amount: *amt}, TokenIn: c.In})
		})
		checkCompoundRow(t, key, c, err, nil, res)
	}
}

func checkCompoundRow(t *testing.T, key string, c goldenRow, err error, out *pool.CalcAmtOutResult, in *pool.CalcAmtInResult) {
	t.Helper()
	if want, ok := changedRows[key]; ok {
		require.Error(t, err, "%s: want %q", key, want)
		require.Equal(t, want, err.Error(), key)
		return
	}
	if c.Err != "" {
		require.Error(t, err, "%s: want legacy error %q", key, c.Err)
		require.Equal(t, c.Err, err.Error(), key)
		return
	}
	require.NoError(t, err, key)
	if out != nil {
		require.Equal(t, c.Dy, out.TokenAmountOut.Amount.String(), key)
	} else {
		require.Equal(t, c.Dy, in.TokenAmountIn.Amount.String(), key)
	}
}

// TestLegacyParityState replays the state walk: quote, then UpdateState, reading reserves after each step.
func TestLegacyParityState(t *testing.T) {
	t.Parallel()
	g := loadCompoundGolden(t)
	var e entity.Pool
	require.NoError(t, json.Unmarshal([]byte(g.Pool), &e))
	p, err := NewPoolSimulator(e)
	require.NoError(t, err)
	c := p.CloneState().(*PoolSimulator)

	var upd []string
	for k := 0; k < 8; k++ {
		amt := uint256.NewInt(uint64(100000 * (k + 1)))
		res, err := quoteParity(func() (*pool.CalcAmtOutResult, error) {
			return p.CalcAmtOut(pool.CalcAmtOutParams{TokenAmountIn: pool.TokenAmt{Token: "bu", Amount: *amt}, TokenOut: "au"})
		})
		if err != nil || res == nil {
			upd = append(upd, "err")
			continue
		}
		p.UpdateState(pool.UpdateStateParams{TokenAmountIn: pool.TokenAmt{Token: "bu", Amount: *amt}, TokenAmountOut: *res.TokenAmountOut})
		for _, r := range p.Info.Reserves {
			upd = append(upd, r.String())
		}
	}
	for _, r := range c.Info.Reserves {
		upd = append(upd, "clone:"+r.String())
	}
	// step 0 changes (decision D / B-prime): legacy wrote [-93114673239022067, 21272421810358792], native rejects the quote
	legacyChanged := append([]string{"err"}, g.Upd[2:]...)
	want := append([]string{"err"}, strings.Join(nil, ""))
	_ = want
	require.Equal(t, legacyChanged, upd[:len(legacyChanged)], "state sequence after the changed step 0")
	require.Equal(t, g.Upd[9:], upd[len(upd)-2:], "clone")
}

func mustCU(t *testing.T, s string) *uint256.Int {
	t.Helper()
	x, err := uint256.FromDecimal(s)
	require.NoError(t, err)
	return x
}
