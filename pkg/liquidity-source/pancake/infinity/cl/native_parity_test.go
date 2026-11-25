package cl

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
	// The golden values are "amount|fee"; the fee is no longer on the native result.
	for k, w := range want {
		want[k], _, _ = strings.Cut(w, "|")
	}

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
		name  string
		chain valueobject.ChainID
	}
	variants := []variant{{"sample", valueobject.ChainID(1)}, {"arc", valueobject.ChainIDArc}}
	const perCase = 32
	r := rand.New(rand.NewPCG(7, 11))
	amounts := make([]string, 0, perCase+5)
	for range perCase {
		exp := 12 + r.IntN(12)
		a := new(big.Int).SetUint64(1 + r.Uint64()%9999)
		a.Mul(a, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(exp)), nil))
		amounts = append(amounts, a.String())
	}
	amounts = append(amounts, "0", "1", "1000000000000000000000000000",
		new(big.Int).Lsh(big.NewInt(1), 255).String(),
		new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1)).String())

	out := map[string]string{}
	for _, v := range variants {
		data, err := os.ReadFile("sample_pool.json")
		require.NoError(t, err)
		var ent entity.Pool
		require.NoError(t, json.Unmarshal(data, &ent))
		sim, err := NewPoolSimulator(ent, v.chain)
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
					out[key] = res.TokenAmountOut.Amount.ToBig().String()
				}
				key = fmt.Sprintf("%s/%d/in/%s", v.name, dir, a)
				resIn, err := sim.CalcAmtIn(pool.CalcAmtInParams{
					TokenAmountOut: pool.TokenAmt{Token: to, Amount: amt}, TokenIn: in})
				if err != nil {
					out[key] = "err:" + err.Error()
				} else {
					out[key] = resIn.TokenAmountIn.Amount.ToBig().String()
				}
			}
		}
	}
	return out
}
