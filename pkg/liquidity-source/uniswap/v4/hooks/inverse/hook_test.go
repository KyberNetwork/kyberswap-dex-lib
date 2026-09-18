package inverse

import (
	"math/big"
	"os"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
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
	t.Helper()
	raw, err := os.ReadFile("testdata/solidity.json")
	require.NoError(t, err)
	var v []vector
	require.NoError(t, json.Unmarshal(raw, &v))
	return v
}
func newHook(s Extra) *Hook {
	return &Hook{BaseHook: uniswapv4.BaseHook{Exchange: "uniswap-v4-inverse"}, State: s}
}
func TestSolidityDifferential(t *testing.T) {
	for _, v := range vectors(t) {
		t.Run(v.Name, func(t *testing.T) {
			h := newHook(v.Before)
			input := v.Input.ToBig()
			saved := new(big.Int).Set(input)
			p := &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: v.ZeroForOne, AmountSpecified: input}
			r, err := h.BeforeSwap(p)
			require.Equal(t, v.Before, h.State, "quote must not mutate its snapshot")
			require.Zero(t, saved.Cmp(input))
			if !v.Success {
				require.Error(t, err, "Solidity reverted: %s", v.Revert)
				return
			}
			require.NoError(t, err)
			require.Equal(t, v.Output.Dec(), new(big.Int).Neg(r.DeltaUnspecified).String())
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
		_, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, AmountSpecified: big.NewInt(1e14)})
		require.Error(t, err)
	}
	h := newHook(s)
	for _, in := range []*big.Int{nil, big.NewInt(-1), big.NewInt(0), new(big.Int).Lsh(big.NewInt(1), 256)} {
		_, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, AmountSpecified: in})
		require.Error(t, err)
	}
	_, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: false, AmountSpecified: big.NewInt(1)})
	require.ErrorIs(t, err, ErrExactOutput)
	h.UpdateBalance(nil)
	require.Equal(t, s, h.State)
	malformed := NewHook(&uniswapv4.HookParam{HookExtra: uniswapv4.HookExtra(`{"bad":`)}).(*Hook)
	_, err = malformed.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, AmountSpecified: big.NewInt(1e14)})
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
		input := new(big.Int).SetUint64(amount)
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
	p := &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: vs[0].ZeroForOne, AmountSpecified: vs[0].Input.ToBig()}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := h.BeforeSwap(p); err != nil {
			b.Fatal(err)
		}
	}
}
