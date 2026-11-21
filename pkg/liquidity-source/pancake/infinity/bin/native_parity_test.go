package bin

import (
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand/v2"
	"os"
	"strings"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Parity guard for the native CalcAmtOut and CalcAmtIn. testdata/native_parity_golden.json holds the amount out
// (or in), fee, and error of the legacy CalcAmountOut and CalcAmountIn, captured before the uint256 port. Every
// case must quote exactly as before, errors included. Keys: <variant>/<dir>/<out|in>/<amount>.
func TestNativeParityGolden(t *testing.T) {
	t.Parallel()
	const golden = "testdata/native_parity_golden.json"
	b, err := os.ReadFile(golden)
	require.NoError(t, err)
	var want map[string]string
	require.NoError(t, json.Unmarshal(b, &want))
	require.NotEmpty(t, want)

	got := nativeParityOutputs(t)
	require.Len(t, got, len(want))
	for k, w := range want {
		require.Equal(t, w, got[k], k)
	}
}

// nativeParityOutputs rebuilds the golden sweep through the native API. The amounts and variants match the
// capture, so the keys line up one to one.
func nativeParityOutputs(t *testing.T) map[string]string {
	type variant struct {
		name    string
		file    string
		hookFee uint32
	}
	variants := []variant{
		{"sample", "sample_pool.json", 0},
		{"thanos-h0", "testdata/thanos_pool.json", 0},
		{"thanos-h777", "testdata/thanos_pool.json", 777},
		{"thanos-h10000", "testdata/thanos_pool.json", 10000},
		{"thanos-h1000000", "testdata/thanos_pool.json", 1000000},
	}
	const perCase = 32
	r := rand.New(rand.NewPCG(7, 11))
	amounts := make([]string, 0, perCase)
	for range perCase {
		exp := 12 + r.IntN(12)
		a := new(big.Int).SetUint64(1 + r.Uint64()%9999)
		a.Mul(a, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(exp)), nil))
		amounts = append(amounts, a.String())
	}

	out := map[string]string{}
	for _, v := range variants {
		data, err := os.ReadFile(v.file)
		require.NoError(t, err)
		var ent entity.Pool
		require.NoError(t, json.Unmarshal(data, &ent))
		if v.hookFee != 0 || strings.HasPrefix(v.name, "thanos") {
			var extra Extra
			require.NoError(t, json.Unmarshal([]byte(ent.Extra), &extra))
			extra.HookSwapFee = v.hookFee
			eb, _ := json.Marshal(extra)
			ent.Extra = string(eb)
		}
		sim, err := NewPoolSimulator(ent, valueobject.ChainIDBSC)
		require.NoError(t, err)
		for dir := range 2 {
			in, to := ent.Tokens[dir].Address, ent.Tokens[1-dir].Address
			for _, a := range amounts {
				var amt uint256.Int
				require.NoError(t, amt.SetFromDecimal(a))
				key := fmt.Sprintf("%s/%d/out/%s", v.name, dir, a)
				res, err := sim.CalcAmtOut(pool.CalcAmtOutParams{
					TokenAmountIn: pool.TokenAmt{Token: in, Amount: amt}, TokenOut: to})
				if err != nil {
					out[key] = "err:" + err.Error()
				} else {
					out[key] = fmt.Sprintf("%s|%s", res.TokenAmountOut.Amount.String(), res.Fee.Amount.String())
				}
				key = fmt.Sprintf("%s/%d/in/%s", v.name, dir, a)
				resIn, err := sim.CalcAmtIn(pool.CalcAmtInParams{
					TokenAmountOut: pool.TokenAmt{Token: to, Amount: amt}, TokenIn: in})
				if err != nil {
					out[key] = "err:" + err.Error()
				} else {
					out[key] = fmt.Sprintf("%s|%s", resIn.TokenAmountIn.Amount.String(), resIn.Fee.Amount.String())
				}
			}
		}
	}
	return out
}
