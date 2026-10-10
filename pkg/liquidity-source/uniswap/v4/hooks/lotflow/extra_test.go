package lotflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
)

// F5: a malformed or schema-skewed hook extra (pool-service input) must never panic an entry point.
// The factory validates the decoded extra structurally and treats an invalid one as untracked.

// tryCall runs f, reporting "ok", "PANIC: ..." or "HANG".
func tryCall(d time.Duration, f func()) string {
	done := make(chan string, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Sprintf("PANIC: %v", r)
				return
			}
			done <- "ok"
		}()
		f()
	}()
	select {
	case r := <-done:
		return r
	case <-time.After(d):
		return "HANG"
	}
}

func TestTryCallInstrumentPositive(t *testing.T) {
	var m *Market
	require.True(t, strings.HasPrefix(tryCall(time.Second, func() { _ = m.Venue }), "PANIC"))
	require.Equal(t, "HANG", tryCall(50*time.Millisecond, func() { time.Sleep(time.Second) }))
}

func hookFromJSON(raw []byte) *Hook {
	return uniswapv4.HookFactories[HookAddresses[0]](&uniswapv4.HookParam{HookExtra: uniswapv4.HookExtra(raw)}).(*Hook)
}

func validExtraJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	h, _ := integrationHook(t, path)
	raw, err := json.Marshal(h.Extra)
	require.NoError(t, err)
	var m map[string]any
	d := json.NewDecoder(strings.NewReader(string(raw)))
	d.UseNumber()
	require.NoError(t, d.Decode(&m))
	return m
}

// entryPoints drives every hook method a pool-service simulator calls; swapInfo is a real SwapInfo
// from a valid quote, fed to UpdateBalance as a route would.
func entryPoints(h *Hook, swapInfo any) map[string]string {
	row := map[string]string{}
	for _, zfo := range []bool{true, false} {
		row[fmt.Sprintf("BeforeSwap zfo=%v", zfo)] = tryCall(10*time.Second, func() {
			_, _ = h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: zfo,
				AmountSpecified: big.NewInt(1_000_000_000)})
		})
		row[fmt.Sprintf("SqrtPriceLimit zfo=%v", zfo)] = tryCall(5*time.Second, func() { h.SqrtPriceLimit(zfo) })
	}
	row["GetReserves"] = tryCall(10*time.Second, func() { _, _ = h.GetReserves(context.Background(), nil) })
	row["CloneState"] = tryCall(2*time.Second, func() { h.CloneState() })
	row["UpdateBalance"] = tryCall(2*time.Second, func() {
		c := h.CloneState().(*Hook)
		c.UpdateBalance(nil)
		c.UpdateBalance(&SwapInfo{res: &SwapResult{}})
		c.UpdateBalance(swapInfo)
	})
	return row
}

func TestMalformedExtrasDoNotPanic(t *testing.T) {
	base := "testdata/fixtures/navjit2-buy-1kusd.json"
	mqPath := "testdata/mq/mq-buy-1kusd.json"
	mut := func(path string, f func(m map[string]any)) []byte {
		m := validExtraJSON(t, path)
		f(m)
		b, err := json.Marshal(m)
		require.NoError(t, err)
		return b
	}
	mk := func(m map[string]any) map[string]any { return m["market"].(map[string]any) }
	c0 := func(m map[string]any) map[string]any { return mk(m)["Constituents"].([]any)[0].(map[string]any) }
	venue := func(m map[string]any) map[string]any { return mk(m)["Venue"].(map[string]any) }
	params := func(m map[string]any) map[string]any { return venue(m)["Params"].(map[string]any) }
	mq := func(m map[string]any) map[string]any { return mk(m)["Mq"].(map[string]any) }
	// AUDIT F5 cases (malformed-extras.json) and the fuzz crasher (FuzzAuditExtra/4ea36ecda39efa57).
	// want untracked: structurally invalid, or a value the chain's setParams / registry cannot hold.
	cases := []struct {
		name      string
		raw       []byte
		untracked bool
	}{
		{"fuzz crasher", []byte(`{"mArket":{}`), true},
		{"case-folded market{}", []byte(`{"mArket":{}}`), true},
		{"market{}", []byte(`{"tracked":true,"trackedAt":9999999999,"market":{}}`), true},
		{"venue{}", []byte(`{"tracked":true,"trackedAt":9999999999,"market":{"Venue":{}}}`), true},
		{"venue nil sqrtP", mut(base, func(m map[string]any) { venue(m)["SqrtPriceX96"] = nil }), true},
		{"pmLot nil", mut(base, func(m map[string]any) { mk(m)["PmLotBalance"] = nil }), true},
		{"unit nil", mut(base, func(m map[string]any) { c0(m)["Unit"] = nil }), true},
		{"nav.unit nil", mut(base, func(m map[string]any) { c0(m)["Nav"].(map[string]any)["Unit"] = nil }), true},
		{"spot nil", mut(base, func(m map[string]any) { c0(m)["Nav"].(map[string]any)["SpotUsd18"] = nil }), true},
		{"pools nil", mut(base, func(m map[string]any) { c0(m)["Pools"] = nil }), true},
		{"sqrtP 2^300", mut(base, func(m map[string]any) {
			venue(m)["SqrtPriceX96"] = json.Number(new(big.Int).Lsh(big.NewInt(1), 300).String())
		}), true},
		{"usdgDec 40", mut(base, func(m map[string]any) { venue(m)["UsdgDecimals"] = 40 }), true},
		{"usdgDec -5", mut(base, func(m map[string]any) { venue(m)["UsdgDecimals"] = -5 }), true},
		{"stockDec 80", mut(base, func(m map[string]any) { c0(m)["Nav"].(map[string]any)["StockDecimals"] = 80 }), true},
		{"width 0", mut(base, func(m map[string]any) { params(m)["WidthTicks"] = 0 }), true},
		{"width 1e7", mut(base, func(m map[string]any) { params(m)["WidthTicks"] = 10_000_000 }), true},
		{"spread>band", mut(base, func(m map[string]any) { params(m)["SellSpreadBps"] = 20_000 }), true},
		{"standing nil liq", mut(base, func(m map[string]any) {
			venue(m)["Standing"] = map[string]any{"Lower": -10, "Upper": 10}
		}), true},
		{"constituents empty", mut(base, func(m map[string]any) { mk(m)["Constituents"] = []any{} }), true},
		{"pool nil entry", mut(base, func(m map[string]any) {
			for k := range c0(m)["Pools"].(map[string]any) {
				c0(m)["Pools"].(map[string]any)[k] = nil
			}
		}), true},
		{"mq conv nil", mut(mqPath, func(m map[string]any) { mq(m)["Conv"] = nil }), true},
		{"mq convTwap nil", mut(mqPath, func(m map[string]any) { mq(m)["ConvTwap"] = nil }), true},
		{"mq ethUsd nil", mut(mqPath, func(m map[string]any) { mq(m)["EthUsd"] = nil }), true},
		{"mq twap card -1", mut(mqPath, func(m map[string]any) {
			mq(m)["ConvTwap"].(map[string]any)["Cardinality"] = -1
		}), true},
		{"mq twap all uninit", mut(mqPath, func(m map[string]any) {
			tw := mq(m)["ConvTwap"].(map[string]any)
			obs := map[string]any{}
			for i := 0; i < 4; i++ {
				obs[fmt.Sprint(i)] = map[string]any{"Timestamp": 1, "TickCumulative": 0, "Initialized": false}
			}
			obs["0"] = map[string]any{"Timestamp": 1, "TickCumulative": 0, "Initialized": true}
			tw["Obs"], tw["Cardinality"], tw["Index"] = obs, 4, 0
		}), false},
	}
	ctrl, c := integrationHook(t, base)
	r, err := ctrl.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: c.Input.ZeroForOne,
		AmountSpecified: c.Input.AmountIn.Copy()})
	require.NoError(t, err, "control quote")
	for name, row := range map[string]map[string]string{"control": entryPoints(hookFromJSON(mut(base,
		func(map[string]any) {})), r.SwapInfo)} {
		for k, v := range row {
			require.Equal(t, "ok", v, "%s %s", name, k)
		}
	}
	bad := 0
	for _, tc := range cases {
		var h *Hook
		res := tryCall(2*time.Second, func() { h = hookFromJSON(tc.raw) })
		require.Equal(t, "ok", res, "%s: factory", tc.name)
		if tc.untracked && h.Tracked {
			bad++
			t.Errorf("%s: invalid extra decoded as tracked", tc.name)
		}
		for k, v := range entryPoints(h, r.SwapInfo) {
			if v != "ok" {
				bad++
				t.Errorf("%-22s %-24s %s", tc.name, k, v)
			}
		}
	}
	require.Zero(t, bad)
}

// Validation accepts every real state: all fixtures and both live venues' Tracked extras.
func TestValidateAcceptsRealMarkets(t *testing.T) {
	var paths []string
	for _, suite := range []string{"fixtures", "a5", "mq", "parity"} {
		paths = append(paths, fixturePaths(t, suite)...)
	}
	n := 0
	for _, p := range paths {
		c, err := LoadCase(p)
		if err != nil || len(c.Pre.Constituents) == 0 {
			continue
		}
		h, _ := integrationHook(t, p)
		require.NoError(t, validateExtra(&h.Extra), p)
		n++
	}
	require.Greater(t, n, 85)
	for _, v := range liveVenues {
		h, raw, err, _ := tapeTrack(t, v, replayFaults{})
		require.NoError(t, err)
		require.NoError(t, validateExtra(&h.Extra), v.name)
		require.True(t, hookFromJSON(raw).Tracked, v.name)
	}
	t.Logf("%d fixture markets and %d live extras validate", n, len(liveVenues))
}

// fuzzEntryPoints is the fuzz body: every entry point on h, reporting "ok", "PANIC: ...", "HANG" or
// "NUMERIC: ..." — a panic the entry points' recover turned into ErrNumeric is still a finding.
func fuzzEntryPoints(h *Hook, amt *big.Int, zfo bool) string {
	return tryCall(20*time.Second, func() {
		numeric := func(err error) {
			if errors.Is(err, ErrNumeric) {
				panic("NUMERIC: " + err.Error())
			}
		}
		r, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: zfo, AmountSpecified: amt})
		numeric(err)
		_, err = h.priceLimit(zfo)
		numeric(err)
		_, err = h.GetReserves(context.Background(), nil)
		numeric(err)
		c := h.CloneState().(*Hook)
		c.UpdateBalance(&SwapInfo{res: &SwapResult{}})
		if err == nil && r != nil {
			c.UpdateBalance(r.SwapInfo)
			for _, z := range []bool{!zfo, zfo} {
				_, err = c.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: z, AmountSpecified: amt})
				numeric(err)
			}
		}
	})
}

// The fuzz instrument registers a positive: a nil Unit planted after the factory panics inside the
// quote, the recover turns it into ErrNumeric, and fuzzEntryPoints still reports it.
func TestFuzzInstrumentPositive(t *testing.T) {
	h := hookFromJSON(validExtraBytes(t, "testdata/fixtures/navjit2-buy-1kusd.json"))
	require.True(t, h.Tracked && h.Market != nil)
	require.Equal(t, "ok", fuzzEntryPoints(h, big.NewInt(1_000_000_000), true))
	h.Market.Constituents[0].Unit = nil
	require.True(t, strings.HasPrefix(fuzzEntryPoints(h, big.NewInt(1_000_000_000), true), "PANIC: NUMERIC"))
}

func validExtraBytes(t testing.TB, path string) []byte {
	c, err := LoadCase(path)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(Extra{Tracked: true, TrackedAt: 9999999999, Lot: c.Pre.Lot.Address,
		Usdg: c.Pre.Hook.Usdg, V3Factory: c.Pre.Hook.V3Factory, Market: marketOfF(c)})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// FuzzHookExtra mutates the raw extra bytes. Go's fuzzer minimizes every new input for up to
// -fuzzminimizetime (60 s default); on these ~10 KB JSON inputs that shows as 0 execs/sec stretches,
// so run it with -fuzzminimizetime 2s. FuzzHookFields reaches the numeric state far faster.
func FuzzHookExtra(f *testing.F) {
	for _, p := range []string{"testdata/fixtures/navjit2-buy-1kusd.json", "testdata/mq/mq-buy-1kusd.json"} {
		f.Add(validExtraBytes(f, p), uint64(1_000_000_000), true)
	}
	f.Add([]byte(`{"mArket":{}`), uint64(999999988), true)
	f.Fuzz(func(t *testing.T, raw []byte, amt uint64, zfo bool) {
		var h *Hook
		if res := tryCall(5*time.Second, func() { h = hookFromJSON(raw) }); res != "ok" {
			t.Fatalf("factory %s on extra %q", res, raw)
		}
		if res := fuzzEntryPoints(h, new(big.Int).SetUint64(amt), zfo); res != "ok" {
			t.Fatalf("%s on extra %q", res, raw)
		}
	})
}

// FuzzHookFields replaces one leaf of a valid extra (a number, its negation, a 200-bit shift, null,
// a deleted key, {} or [] or a bool) and drives every entry point, including UpdateBalance with a real
// SwapInfo. AUDIT2's FuzzAudit2Fields, which found F-N1 (corpus 0db89d8b5acdaf40, kept as a seed).
func FuzzHookFields(f *testing.F) {
	type parsed struct {
		root   any
		leaves [][]any
	}
	var ps []parsed
	for _, p := range []string{"testdata/fixtures/navjit2-buy-1kusd.json", "testdata/mq/mq-buy-1kusd.json",
		"testdata/a5/amznaapl-buy-1kusd-standing.json"} {
		root := decodeNumbers(f, validExtraBytes(f, p))
		var ls [][]any
		jsonLeaves(root, nil, &ls)
		ps = append(ps, parsed{root, ls})
	}
	f.Add(uint8(0), uint32(5), uint8(0), []byte{1}, uint64(1_000_000_000), true)
	f.Add(uint8(1), uint32(77), uint8(3), []byte{0xff, 0xff}, uint64(1_000_000), false)
	f.Add(uint8(2), uint32(200), uint8(1), []byte{0x80}, uint64(10_000_000_000), true)
	f.Fuzz(func(t *testing.T, which uint8, leaf uint32, mode uint8, val []byte, amt uint64, zfo bool) {
		p := ps[int(which)%len(ps)]
		b, _ := json.Marshal(p.root)
		root := decodeNumbers(t, b)
		path := p.leaves[int(leaf)%len(p.leaves)]
		n := new(big.Int).SetBytes(val)
		switch mode % 8 {
		case 0:
			jsonSet(root, path, json.Number(n.String()), false)
		case 1:
			jsonSet(root, path, json.Number(new(big.Int).Neg(n).String()), false)
		case 2:
			jsonSet(root, path, nil, false)
		case 3:
			jsonSet(root, path, nil, true)
		case 4:
			jsonSet(root, path, json.Number(new(big.Int).Lsh(n, 200).String()), false)
		case 5:
			jsonSet(root, path, map[string]any{}, false)
		case 6:
			jsonSet(root, path, []any{}, false)
		case 7:
			jsonSet(root, path, n.Sign() != 0, false)
		}
		raw, _ := json.Marshal(root)
		var h *Hook
		if res := tryCall(5*time.Second, func() { h = hookFromJSON(raw) }); res != "ok" {
			t.Fatalf("factory %s on leaf %v mode %d val %x", res, path, mode%8, val)
		}
		if res := fuzzEntryPoints(h, new(big.Int).SetUint64(amt), zfo); res != "ok" {
			t.Fatalf("%s on leaf %v mode %d val %x amt %d zfo %v", res, path, mode%8, val, amt, zfo)
		}
	})
}

func decodeNumbers(t testing.TB, b []byte) any {
	var root any
	d := json.NewDecoder(strings.NewReader(string(b)))
	d.UseNumber()
	if err := d.Decode(&root); err != nil {
		t.Fatal(err)
	}
	return root
}

// jsonLeaves lists the path of every value under v, keys in sorted order.
func jsonLeaves(v any, prefix []any, out *[][]any) {
	switch x := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			p := append(append([]any{}, prefix...), k)
			*out = append(*out, p)
			jsonLeaves(x[k], p, out)
		}
	case []any:
		for i := range x {
			p := append(append([]any{}, prefix...), i)
			*out = append(*out, p)
			jsonLeaves(x[i], p, out)
		}
	}
}

// jsonSet sets (or deletes) the value at path.
func jsonSet(root any, path []any, val any, del bool) {
	cur := root
	for i, k := range path {
		last := i == len(path)-1
		switch c := cur.(type) {
		case map[string]any:
			if last {
				if del {
					delete(c, k.(string))
				} else {
					c[k.(string)] = val
				}
				return
			}
			cur = c[k.(string)]
		case []any:
			if last {
				c[k.(int)] = val
				return
			}
			cur = c[k.(int)]
		default:
			return
		}
	}
}

// marketOfF is marketOf without a test (fuzz seeding).
func marketOfF(c *Case) *Market {
	var m *Market
	func() {
		defer func() { _ = recover() }()
		m = marketOf(&testing.T{}, c)
	}()
	return m
}

// decodeExtra's recover has a real reproducer, but only in sequence: goccy/go-json v0.10.5 decodes from
// a pooled buffer, and on a short truncated input with an escape inside a key (FuzzHookExtra corpus
// 8e6fd1838d839f3e, `{"mArket":{"Mq\2`) it can index past the input into what a previous, longer
// document left there: index out of range in skipWhiteSpace (internal/decoder/context.go:49). In
// a fresh process the input decodes to an error; after a ~10 KB extra it panics in a fraction of runs
// (so it is not asserted alone here: earlier tests in the binary leave long documents in the pool). The
// loop counts the panics of the raw decoder (the instrument must fire) and requires decodeExtra to
// return an error every time. Under -race the raw decoder did not panic in 600 runs (go1.26.2), so the
// instrument is required to fire only without it.
func TestDecodeExtraRecoversDecoderPanic(t *testing.T) {
	long := validExtraBytes(t, "testdata/fixtures/navjit2-buy-1kusd.json")
	short := []byte("{\"mArket\":{\"Mq\\2")
	raw := func(b []byte) (panicked bool) {
		defer func() { panicked = recover() != nil }()
		_ = uniswapv4.HookExtra(b).Unmarshal(&Extra{})
		return false
	}
	panics := 0
	for i := 0; i < 200; i++ {
		var x Extra
		require.NoError(t, decodeExtra(long, &x))
		if raw(short) {
			panics++
		}
		require.NoError(t, decodeExtra(long, &x))
		require.NotPanics(t, func() { require.Error(t, decodeExtra(short, &Extra{})) })
		require.False(t, hookFromJSON(short).Tracked)
	}
	t.Logf("raw decoder panics after a long extra: %d/200 (race build %v)", panics, raceBuild)
	if !raceBuild {
		require.Positive(t, panics, "the reproducer must reproduce")
	}
}
