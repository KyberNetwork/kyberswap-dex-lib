package stableng

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type probeDiff struct{ Key, Legacy, Native string }

// TestProbeClassify replays the 4000-input probe golden through the native surface, one fresh pool per input,
// and counts each row's class against legacy. It writes the per-row results to CLASSIFY_OUT when set.
func TestProbeClassify(t *testing.T) {
	path := os.Getenv("PROBE_GOLDEN")
	if path == "" {
		t.Skip("set PROBE_GOLDEN=<path to legacy_goldens_4000_probe.json>")
	}
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	var golden []goldenPool
	require.NoError(t, json.Unmarshal(b, &golden))

	counts := map[string]int{}
	var diffs []probeDiff
	for pi, gp := range golden {
		var e entity.Pool
		require.NoError(t, json.Unmarshal([]byte(gp.Pool), &e))
		toks := []string{}
		for _, tk := range e.Tokens {
			toks = append(toks, tk.Address)
		}
		for idx, c := range gp.Quotes {
			s, err := NewPoolSimulator(e)
			require.NoError(t, err)
			var amt uint256.Int
			require.NoError(t, amt.SetFromDecimal(c.Amount))
			res, nerr := s.CalcAmtOut(pool.CalcAmtOutParams{TokenAmountIn: pool.TokenAmt{Token: c.In, Amount: amt}, TokenOut: c.Out})
			classify(counts, &diffs, fmt.Sprintf("%d/Quote/%d", pi, idx), c, nerr, res, nil)
		}
		for idx, c := range gp.Ins {
			s, err := NewPoolSimulator(e)
			require.NoError(t, err)
			var amt uint256.Int
			require.NoError(t, amt.SetFromDecimal(c.Amount))
			res, nerr := s.CalcAmtIn(pool.CalcAmtInParams{TokenAmountOut: pool.TokenAmt{Token: c.Out, Amount: amt}, TokenIn: c.In})
			classify(counts, &diffs, fmt.Sprintf("%d/In/%d", pi, idx), c, nerr, nil, res)
		}
	}
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("CLASS %-60s %d\n", k, counts[k])
	}
	if out := os.Getenv("CLASSIFY_OUT"); out != "" {
		j, _ := json.MarshalIndent(diffs, "", " ")
		require.NoError(t, os.WriteFile(out, j, 0o644))
	}
}

func classify(counts map[string]int, diffs *[]probeDiff, key string, c goldenRow, nerr error, out *pool.CalcAmtOutResult, in *pool.CalcAmtInResult) {
	legacy := c.Dy
	if c.Err != "" {
		legacy = "ERR:" + c.Err
	}
	native := ""
	switch {
	case nerr != nil:
		native = "ERR:" + nerr.Error()
	case out != nil:
		native = out.TokenAmountOut.Amount.Dec()
	default:
		native = in.TokenAmountIn.Amount.Dec()
	}
	var class string
	switch {
	case legacy == native:
		class = "identical"
	case c.Err == "" && nerr != nil && native == "ERR:negative result":
		class = "negative-reserve"
	case c.Err != "" && nerr == nil:
		class = "legacy-err-native-value"
	case c.Err != "" && nerr != nil:
		class = "both-err-text-differs"
	default:
		class = "value-differs"
	}
	counts[class]++
	if class != "identical" {
		*diffs = append(*diffs, probeDiff{key, legacy, native})
	}
}
