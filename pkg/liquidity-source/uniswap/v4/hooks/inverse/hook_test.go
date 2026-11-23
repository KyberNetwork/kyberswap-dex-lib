package inverse

import (
	"math/big"
	"os"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

type vector struct {
	Name       string      `json:"name"`
	Before     Extra       `json:"before"`
	After      Extra       `json:"after"`
	ZeroForOne bool        `json:"zeroForOne"`
	Input      uint256.Int `json:"amountIn"`
	Output     uint256.Int `json:"amountOut"`
	Success    bool        `json:"success"`
	Revert     string      `json:"revertData"`
}

func vectors(t *testing.T) []vector {
	return loadVectors(t, "testdata/solidity.json")
}
func loadVectors(t *testing.T, path string) []vector {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var v []vector
	require.NoError(t, json.Unmarshal(raw, &v))
	return v
}
func newHook(s Extra) *Hook {
	return &Hook{Exchange: "uniswap-v4-inverse", State: s}
}
func TestSolidityDifferential(t *testing.T) {
	testDifferential(t, vectors(t))
}
func TestMinedRobinhoodDifferential(t *testing.T) {
	testDifferential(t, loadVectors(t, "testdata/robinhood.json"))
}
func testDifferential(t *testing.T, vs []vector) {
	t.Helper()
	for _, v := range vs {
		t.Run(v.Name, func(t *testing.T) {
			h := newHook(v.Before)
			input := v.Input
			saved := input
			p := &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: v.ZeroForOne, AmountSpecified: &input}
			r, err := h.BeforeSwap(p)
			require.Equal(t, v.Before, h.State, "quote must not mutate its snapshot")
			require.Equal(t, saved, input)
			if !v.Success {
				require.Error(t, err, "Solidity reverted: %s", v.Revert)
				return
			}
			require.NoError(t, err)
			require.Equal(t, v.Output.Dec(), new(uint256.Int).Neg(r.DeltaUnspecified).Dec())
			require.Equal(t, v.After, r.SwapInfo.(SwapInfo).After)
			clone := h.CloneState().(*Hook)
			clone.UpdateBalance(r.SwapInfo)
			require.Equal(t, v.After, clone.State)
			require.Equal(t, v.Before, h.State)
			h.UpdateBalance(r.SwapInfo)
			require.Equal(t, v.After, h.State)
			h.UpdateBalance(r.SwapInfo)
			require.Equal(t, v.After, h.State, "stale update must not be applied twice")
		})
	}
}
func TestFailClosed(t *testing.T) {
	s := vectors(t)[0].Before
	for _, mutate := range []func(*Extra){func(s *Extra) { s.Live = false }, func(s *Extra) { s.Version = 0 }, func(s *Extra) { s.Index.Clear() }, func(s *Extra) { s.ReserveShares.Clear() }, func(s *Extra) { s.ProtocolFee = 1001 }, func(s *Extra) { s.BlockNumber = 0 }, func(s *Extra) { s.Liquidity.Clear() }, func(s *Extra) { s.SqrtPriceX96.Clear() }} {
		c := s
		mutate(&c)
		h := newHook(c)
		for _, calcOut := range []bool{true, false} {
			_, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: calcOut, AmountSpecified: uint256.NewInt(1e14)})
			require.Error(t, err)
		}
	}
	h := newHook(s)
	for _, in := range []*uint256.Int{nil, new(uint256.Int).Neg(uint256.NewInt(1)), uint256.NewInt(0)} {
		_, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, AmountSpecified: in})
		require.Error(t, err)
	}
	for _, in := range []*uint256.Int{nil, new(uint256.Int).Neg(uint256.NewInt(1)), uint256.NewInt(0)} {
		_, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: false, AmountSpecified: in})
		require.Error(t, err)
	}
	h.UpdateBalance(nil)
	require.Equal(t, s, h.State)
	malformed := NewHook(&uniswapv4.HookParam{HookExtra: uniswapv4.HookExtra(`{"bad":`)}).(*Hook)
	_, err := malformed.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, AmountSpecified: uint256.NewInt(1e14)})
	require.Error(t, err)
}

func FuzzQuoteDoesNotMutate(f *testing.F) {
	raw, err := os.ReadFile("testdata/solidity.json")
	if err != nil {
		f.Fatal(err)
	}
	var vs []vector
	if err = json.Unmarshal(raw, &vs); err != nil {
		f.Fatal(err)
	}
	f.Add(uint64(1e14), uint8(0), false)
	f.Add(uint64(0), uint8(5), true)
	f.Add(^uint64(0), uint8(100), true)
	f.Fuzz(func(t *testing.T, amount uint64, pick uint8, large bool) {
		v := vs[int(pick)%len(vs)]
		h := newHook(v.Before)
		input := uint256.NewInt(amount)
		if large {
			input.Lsh(input, 70)
		}
		params := &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: v.ZeroForOne, AmountSpecified: input}
		r, err := h.BeforeSwap(params)
		require.Equal(t, v.Before, h.State)
		r2, err2 := h.BeforeSwap(params)
		require.Equal(t, err, err2)
		require.Equal(t, r, r2)
		if err == nil {
			require.Negative(t, r.DeltaUnspecified.Sign())
			require.NoError(t, r.SwapInfo.(SwapInfo).After.validate())
		}
	})
}
func BenchmarkQuote(b *testing.B) {
	raw, _ := os.ReadFile("testdata/solidity.json")
	var vs []vector
	_ = json.Unmarshal(raw, &vs)
	h := newHook(vs[0].Before)
	p := &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: vs[0].ZeroForOne, AmountSpecified: &vs[0].Input}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := h.BeforeSwap(p); err != nil {
			b.Fatal(err)
		}
	}
}

// CalcIn reverses exact-in: for each Solidity exact-in vector, the min amountIn reaching its output is at most its
// input, and 1 wei less falls short.
func TestCalcInReversesSolidityVectors(t *testing.T) {
	for _, v := range vectors(t) {
		if !v.Success {
			continue
		}
		output := v.Output
		r, err := newHook(v.Before).BeforeSwap(&uniswapv4.BeforeSwapParams{
			CalcOut: false, ZeroForOne: v.ZeroForOne, AmountSpecified: &output,
		})
		require.NoError(t, err, v.Name)
		amountIn := big256.ToBig(r.DeltaUnspecified)
		require.LessOrEqual(t, amountIn.Cmp(v.Input.ToBig()), 0, v.Name)
		require.Equal(t, new(uint256.Int).Neg(&v.Output), r.DeltaSpecified, v.Name)

		out, _, err := quote(v.Before, v.ZeroForOne, amountIn)
		require.NoError(t, err, v.Name)
		require.GreaterOrEqual(t, out.Cmp(v.Output.ToBig()), 0, v.Name)
		if out, _, err = quote(v.Before, v.ZeroForOne, new(big.Int).Sub(amountIn, big.NewInt(1))); err == nil {
			require.Negative(t, out.Cmp(v.Output.ToBig()), "%s: amountIn %s is not minimal", v.Name, amountIn)
		}
	}
}
