package flywheelfun

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// parityRow is one golden line. Field names match testdata/parity/golden.json.
type parityRow struct {
	Case      string `json:"case"`
	TokenOut  string `json:"tokenOut"`
	Amount    string `json:"amount"`
	Remaining string `json:"remaining"`
	Fee       string `json:"fee"`
	FeeToken  string `json:"feeToken"`
	Gas       int64  `json:"gas"`
	Valid     bool   `json:"valid"`
	Err       string `json:"err"`
}

func parityRowOf(name, tokenOut string, r *pool.CalcAmtOutResult, err error, valid bool) parityRow {
	row := parityRow{Case: name, TokenOut: tokenOut, Valid: valid}
	if err != nil {
		row.Err = err.Error()
		return row
	}
	if r == nil {
		return row
	}
	row.Amount = r.TokenAmountOut.Amount.Dec()
	row.Remaining = r.RemainingTokenAmountIn.Amount.Dec()
	row.Gas = r.Gas
	return row
}

// TestParityGolden replays the golden quote sequence on the uint256 surface and
// requires every row to match testdata/parity/golden.json exactly.
func TestParityGolden(t *testing.T) {
	b, err := os.ReadFile("testdata/parity/golden.json")
	require.NoError(t, err)
	var want []parityRow
	require.NoError(t, json.Unmarshal(b, &want))
	// The fee is no longer on the native result: drop the fee rows and the fee columns.
	want = slices.DeleteFunc(want, func(r parityRow) bool { return strings.HasSuffix(r.Case, "/fee") })
	for i := range want {
		want[i].Fee, want[i].FeeToken = "", ""
	}

	var got []parityRow
	for i, p := range snapshots(t) {
		name := fmt.Sprintf("snap%d", i)
		s := newSim(t, p)
		for _, amt := range []uint64{1000000, 1000000000, 10000000000} {
			in := pool.TokenAmt{Token: WETH, Amount: *uint256.NewInt(amt)}
			r, err := s.CalcAmtOut(pool.CalcAmtOutParams{TokenAmountIn: in, TokenOut: s.Info.Address})
			got = append(got, parityRowOf(fmt.Sprintf("%s/buy/%d", name, amt), s.Info.Address, r, err, s.Valid))
			if err != nil {
				continue
			}
			sellIn := pool.TokenAmt{Token: s.Info.Address, Amount: r.TokenAmountOut.Amount}
			sell, serr := s.CalcAmtOut(pool.CalcAmtOutParams{TokenAmountIn: sellIn, TokenOut: WETH})
			got = append(got, parityRowOf(fmt.Sprintf("%s/sell/%d", name, amt), WETH, sell, serr, s.Valid))
			if amt == 1000000000 {
				clone := s.CloneState().(*PoolSimulator)
				s.UpdateState(pool.UpdateStateParams{TokenAmountIn: in, TokenAmountOut: *r.TokenAmountOut, SwapInfo: r.SwapInfo})
				next, nerr := s.CalcAmtOut(pool.CalcAmtOutParams{TokenAmountIn: in, TokenOut: s.Info.Address})
				got = append(got, parityRowOf(name+"/after-update", s.Info.Address, next, nerr, s.Valid))
				clone.UpdateState(pool.UpdateStateParams{TokenAmountIn: in, TokenAmountOut: *r.TokenAmountOut, SwapInfo: r.SwapInfo})
				got = append(got, parityRowOf(name+"/clone-update", s.Info.Address, nil, nil, clone.Valid))
				s.UpdateState(pool.UpdateStateParams{TokenAmountIn: in, TokenAmountOut: *r.TokenAmountOut, SwapInfo: r.SwapInfo})
				got = append(got, parityRowOf(name+"/stale-replay", s.Info.Address, nil, nil, s.Valid))
			}
		}
	}
	require.Equal(t, want, got)
}
