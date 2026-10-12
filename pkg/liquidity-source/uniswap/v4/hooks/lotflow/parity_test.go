package lotflow

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

// Live parity (A7 part 2), MEASURED-ON-FORK. Needs a running local anvil with V17 + the MQ stack deployed by
// testdata/tools/parity.py; LOTFLOW_PARITY = path to its deploy.json (skipped otherwise).
// For each venue (bAMZNAAPL, LOT-Mag 5, the MQ basket) and side, 100 log-uniform random sizes are quoted
// by the simulator (state loaded through RPCLoader + Tracker) and by the on-fork V4Quoter at the same
// state. Over-quote = QuoteConservative fills where the quoter reverts, or returns more. A sell the quoter
// refuses for float (it does not prefund) is compared again prefunded: simulator Prefund=true vs the
// quoter with the PoolManager's LOT balance overridden to float + amountIn (what a prefunded swap sees).

type parityDeploy struct {
	Anvil  string `json:"anvil"`
	PM     string `json:"pm"`
	Quoter string `json:"quoter"`
	Pin    int64  `json:"pin"`
	Venues map[string]struct {
		Lot    string `json:"lot"`
		Hook   string `json:"hook"`
		PoolID string `json:"poolId"`
	} `json:"venues"`
}

type paritySample struct {
	Venue, Side       string
	AmountIn          string
	Sim, Cons, Quoter string // amountOut or "revert:<name>"
	QuoterRevert      string
	Prefund           bool
	DevRaw            string
	DevBps            float64
	Over              bool
	Inexact           bool
	Note              string
}

func TestLiveParity(t *testing.T) {
	path := os.Getenv("LOTFLOW_PARITY")
	if path == "" {
		t.Skip("LOTFLOW_PARITY not set (live anvil parity; see testdata/tools/parity.py)")
	}
	var d parityDeploy
	require.NoError(t, readJSON(path, &d)) // testdata/parity/deploy.json is stored gzipped
	require.True(t, strings.HasPrefix(d.Anvil, "http://127.0.0.1:"), "local anvil only")
	ctx := context.Background()
	rc, err := rpc.DialContext(ctx, d.Anvil)
	require.NoError(t, err)

	var all []paritySample
	names := []string{"two", "mag5", "mq"}
	for vi, name := range names {
		v := d.Venues[name]
		cfg := DefaultConfig("", "", FinalisedRules)
		l, err := NewRPCLoader(ctx, d.Anvil, v.Hook, d.PM, cfg)
		require.NoError(t, err, name)
		tr := &Tracker{Cfg: cfg, Hook: v.Hook, PoolManager: d.PM, NavGuard: l.NavGuard, Registry: l.Registry,
			VenueBook: l.VenueBook}
		st, err := tr.Load(ctx, l, v.Lot)
		require.NoError(t, err, name)
		require.Equal(t, strings.ToLower(v.PoolID), st.VenuePoolID, name)
		usdgIs0 := st.Market.Venue.UsdgIs0
		lotSlot := lotBalanceSlot(t, ctx, rc, v.Lot, d.PM)

		rng := rand.New(rand.NewSource(20261009 + int64(vi)))
		type job struct {
			buy bool
			amt *big.Int
		}
		var jobs []job
		for _, buy := range []bool{true, false} {
			for k := 0; k < 100; k++ {
				// log-uniform: buys $1 .. $300k of USDG; sells 1 .. 300k LOT
				x := math.Pow(10, rng.Float64()*math.Log10(3e5))
				amt := new(big.Float).Mul(big.NewFloat(x), big.NewFloat(1e6))
				if !buy {
					amt.Mul(big.NewFloat(x), big.NewFloat(1e18))
				}
				ai, _ := amt.Int(nil)
				jobs = append(jobs, job{buy, ai})
			}
		}
		res := make([]paritySample, len(jobs))
		var wg sync.WaitGroup
		sem := make(chan struct{}, 4)
		for i, j := range jobs {
			wg.Add(1)
			sem <- struct{}{}
			go func(i int, j job) {
				defer wg.Done()
				defer func() { <-sem }()
				res[i] = parityOne(ctx, rc, &d, name, v.Lot, v.PoolID, v.Hook, usdgIs0, lotSlot, cfg, st.Market, j.buy, j.amt)
			}(i, j)
		}
		wg.Wait()
		all = append(all, res...)
	}

	// summary
	type agg struct {
		N, BothFill, BothRevert, SimRevertOnly, Over, Exact, Inexact int
		MaxDevBps, MaxDevBpsInexact                                  float64
	}
	sum := map[string]*agg{}
	for _, s := range all {
		k := s.Venue + ":" + s.Side
		if s.Prefund {
			k += ":prefund"
		}
		a := sum[k]
		if a == nil {
			a = &agg{}
			sum[k] = a
		}
		a.N++
		simOK, qOK := !strings.HasPrefix(s.Sim, "revert"), !strings.HasPrefix(s.Quoter, "revert")
		switch {
		case simOK && qOK:
			a.BothFill++
			if s.DevRaw == "0" {
				a.Exact++
			}
			dv := math.Abs(s.DevBps)
			if s.Inexact {
				a.MaxDevBpsInexact = math.Max(a.MaxDevBpsInexact, dv)
			} else {
				a.MaxDevBps = math.Max(a.MaxDevBps, dv)
			}
		case !simOK && !qOK:
			a.BothRevert++
		case !simOK && qOK:
			a.SimRevertOnly++
		}
		if s.Over {
			a.Over++
		}
		if s.Inexact {
			a.Inexact++
		}
	}
	keys := make([]string, 0, len(sum))
	for k := range sum {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	over := 0
	for _, k := range keys {
		a := sum[k]
		over += a.Over
		t.Logf("%-18s n=%3d bothFill=%3d (exact %3d) bothRevert=%3d simRevertOnly=%d over=%d inexact=%d "+
			"maxDevBps=%.4f (inexact %.4f)", k, a.N, a.BothFill, a.Exact, a.BothRevert, a.SimRevertOnly, a.Over,
			a.Inexact, a.MaxDevBps, a.MaxDevBpsInexact)
	}
	if out := os.Getenv("LOTFLOW_PARITY_OUT"); out != "" {
		b, _ := json.MarshalIndent(map[string]any{"pin": d.Pin, "samples": all, "summary": sum}, "", " ")
		require.NoError(t, os.WriteFile(out, b, 0o644))
	}
	for _, s := range all {
		if s.Over {
			t.Errorf("OVER-QUOTE %s %s %s: conservative %s, quoter %s %s", s.Venue, s.Side, s.AmountIn, s.Cons,
				s.Quoter, s.QuoterRevert)
		}
	}
	require.Zero(t, over, "over-quotes")
}

func parityOne(ctx context.Context, rc *rpc.Client, d *parityDeploy, venue, lot, poolID, hook string, usdgIs0 bool,
	lotSlot common.Hash, cfg *Config, m *Market, buy bool, amt *big.Int) paritySample {
	_ = poolID
	side := map[bool]string{true: "buy", false: "sell"}[buy]
	s := paritySample{Venue: venue, Side: side, AmountIn: amt.String()}
	zfo := buy == usdgIs0
	in := SwapInput{BuyLot: buy, AmountIn: amt}
	sim, serr := Quote(cfg, m, in)
	cons, cerr := QuoteConservative(cfg, m, in)
	q, qrev, qerr := quoteV4(ctx, rc, d.Quoter, lot, hook, usdgIs0, zfo, amt, nil)
	fill := func(r *SwapResult, err error) string {
		if err != nil {
			var n NamedError
			if errors.As(err, &n) {
				return "revert:" + n.ErrorName()
			}
			return "revert:" + err.Error()
		}
		return r.AmountOut.String()
	}
	compare := func() {
		s.Sim, s.Cons = fill(sim, serr), fill(cons, cerr)
		s.Inexact = sim != nil && sim.Inexact
		if qerr != nil {
			s.Quoter, s.QuoterRevert = "revert", hexOf(qrev)
			if serr != nil {
				if sel := Selector(serr); sel != "" && !bytes.Contains(qrev, common.FromHex(sel)) {
					s.Note = "both revert, different selector " + sel
				}
			}
			s.Over = cerr == nil // the simulator would fill a swap the chain refuses
			return
		}
		s.Quoter = q.String()
		if serr == nil {
			dev := new(big.Int).Sub(sim.AmountOut, q)
			s.DevRaw = dev.String()
			f, _ := new(big.Float).Quo(new(big.Float).SetInt(dev), new(big.Float).SetInt(q)).Float64()
			s.DevBps = f * 1e4
		}
		s.Over = cerr == nil && cons.AmountOut.Cmp(q) > 0
	}
	compare()
	if !buy && qerr != nil && bytes.Contains(qrev, sel("SellExceedsFloat(uint256,uint256)")) {
		// prefunded comparison: the quoter with PM LOT float + amountIn (what a prefunded swap sees at swap
		// time), the simulator with Prefund
		pre := s
		cur, err := balanceOf(ctx, rc, lot, d.PM)
		if err != nil {
			s.Note = "prefund: " + err.Error()
			return s
		}
		ov, err := backedFloatOverride(ctx, rc, lot, d.PM, lotSlot, cur, amt, m)
		if err != nil {
			s.Note = "prefund override: " + err.Error()
			return s
		}
		in.Prefund = true
		sim, serr = Quote(cfg, m, in)
		cons, cerr = QuoteConservative(cfg, m, in)
		q, qrev, qerr = quoteV4(ctx, rc, d.Quoter, lot, hook, usdgIs0, zfo, amt, ov)
		s = paritySample{Venue: venue, Side: side, AmountIn: amt.String(), Prefund: true}
		compare()
		if pre.Over {
			s.Over = true
		}
		s.Note = strings.TrimSpace(s.Note + " (pay-after: both SellExceedsFloat)")
	}
	return s
}

// quoteV4 is V4Quoter.quoteExactInputSingle(((c0,c1,0,1,hook),zfo,amt,0x)) by eth_call, 30M gas.
func quoteV4(ctx context.Context, rc *rpc.Client, quoter, lot, hook string, usdgIs0, zfo bool, amt *big.Int,
	override map[string]any) (*big.Int, []byte, error) {
	c0, c1 := lotUsdg(lot, usdgIs0)
	b2i := map[bool]int64{true: 1, false: 0}
	data := append(sel("quoteExactInputSingle(((address,address,uint24,int24,address),bool,uint128,bytes))"),
		concat([][]byte{encWord(big.NewInt(32)), encAddr(c0), encAddr(c1), encWord(big.NewInt(0)), encWord(big.NewInt(1)),
			encAddr(hook), encWord(big.NewInt(b2i[zfo])), encWord(amt), encWord(big.NewInt(256)), encWord(big.NewInt(0))})...)
	msg := map[string]any{"to": quoter, "data": hexutil.Bytes(data), "gas": hexutil.Uint64(30_000_000)}
	var out hexutil.Bytes
	args := []any{msg, parityBlock}
	if override != nil {
		args = append(args, override)
	}
	if err := retryLimited(ctx, func() error { return rc.CallContext(ctx, &out, "eth_call", args...) }); err != nil {
		var de rpc.DataError
		if errors.As(err, &de) {
			if s, ok := de.ErrorData().(string); ok {
				return nil, common.FromHex(s), err
			}
		}
		return nil, nil, err
	}
	return wordAt(out, 0), nil, nil
}

var parityUsdg = "0x5fc5360D0400a0Fd4f2af552ADD042D716F1d168"

func lotUsdg(lot string, usdgIs0 bool) (string, string) {
	if usdgIs0 {
		return parityUsdg, lot
	}
	return lot, parityUsdg
}

func balanceOf(ctx context.Context, rc *rpc.Client, token, who string) (*big.Int, error) {
	var out hexutil.Bytes
	msg := map[string]any{"to": token, "data": hexutil.Bytes(append(sel("balanceOf(address)"), encAddr(who)...))}
	if err := rc.CallContext(ctx, &out, "eth_call", msg, parityBlock); err != nil {
		return nil, err
	}
	return wordAt(out, 0), nil
}

// parityBlock is the block tag the quoter / slot-probe helpers read at (the live test pins one block).
var parityBlock = "latest"

// backedFloatOverride is the state a PREFUNDED sell sees at swap time, as an eth_call state override: the
// seller's LOT (minted earlier, so backed) is already in the PoolManager. LOT balanceOf(PM) += amt, LOT
// totalSupply += amt, and each constituent balanceOf(basket) += ceil(unit*amt/1e18) (the mint that created
// it). An override of the PM balance alone is unbacked: the hook's redeem then fails ERC20InsufficientBalance
// on a constituent transfer (measured in the first parity run).
func backedFloatOverride(ctx context.Context, rc *rpc.Client, lot, pm string, lotSlot common.Hash, cur, amt *big.Int,
	m *Market) (map[string]any, error) {
	diffs := map[string]map[string]string{lot: {lotSlot.Hex(): common.BigToHash(new(big.Int).Add(cur, amt)).Hex()}}
	tsSlot, ts, err := findSlot(ctx, rc, lot, sel("totalSupply()"), nil, func(base common.Hash) common.Hash { return base })
	if err != nil {
		return nil, fmt.Errorf("totalSupply slot: %w", err)
	}
	diffs[lot][tsSlot.Hex()] = common.BigToHash(new(big.Int).Add(ts, amt)).Hex()
	units := make([]*big.Int, len(m.Constituents))
	for i, c := range m.Constituents {
		units[i] = c.Unit
	}
	for i, need := range MintAmounts(units, amt) {
		tok := strings.ToLower(m.Constituents[i].Stock)
		slot, bal, err := findSlot(ctx, rc, tok, append(sel("balanceOf(address)"), encAddr(lot)...), nil,
			func(base common.Hash) common.Hash { return crypto.Keccak256Hash(encAddr(lot), base.Bytes()) })
		if err != nil {
			return nil, fmt.Errorf("balance slot of %s: %w", tok, err)
		}
		if diffs[tok] == nil {
			diffs[tok] = map[string]string{}
		}
		diffs[tok][slot.Hex()] = common.BigToHash(new(big.Int).Add(bal, need)).Hex()
	}
	ov := map[string]any{}
	for a, d := range diffs {
		ov[a] = map[string]any{"stateDiff": d}
	}
	return ov, nil
}

// erc7201ERC20 is OpenZeppelin's namespaced ERC20 storage base (upgradeable tokens); the balance mapping
// is at +0 and totalSupply at +2.
var erc7201ERC20 = common.HexToHash("0x52c63247e1f47db19d5ce0460030c497f067ca4cebf71ba98eeadabe20bace00")

// findSlot locates the storage slot a view reads by overriding candidates (plain bases 0..11, then the
// namespaced layout) with a marker and checking the view returns it. at maps a mapping base to the slot.
func findSlot(ctx context.Context, rc *rpc.Client, token string, call []byte, _ any,
	at func(base common.Hash) common.Hash) (common.Hash, *big.Int, error) {
	key := token + hexOf(call)
	if c, ok := slotCache.Load(key); ok { // state is fixed for the run (every call is an eth_call)
		e := c.([2]any)
		return e[0].(common.Hash), e[1].(*big.Int), nil
	}
	slot, cur, err := findSlotUncached(ctx, rc, token, call, at)
	if err == nil {
		slotCache.Store(key, [2]any{slot, cur})
	}
	return slot, cur, err
}

var slotCache sync.Map

func findSlotUncached(ctx context.Context, rc *rpc.Client, token string, call []byte,
	at func(base common.Hash) common.Hash) (common.Hash, *big.Int, error) {
	view := func(ov map[string]any) (*big.Int, error) {
		var out hexutil.Bytes
		args := []any{map[string]any{"to": token, "data": hexutil.Bytes(call)}, parityBlock}
		if ov != nil {
			args = append(args, ov)
		}
		err := retryLimited(ctx, func() error { return rc.CallContext(ctx, &out, "eth_call", args...) })
		return wordAt(out, 0), err
	}
	cur, err := view(nil)
	if err != nil {
		return common.Hash{}, nil, err
	}
	marker := big.NewInt(0x1234567)
	var bases []common.Hash
	for b := int64(0); b < 12; b++ {
		bases = append(bases, common.BigToHash(big.NewInt(b)))
	}
	bases = append(bases, erc7201ERC20, slotPlus(erc7201ERC20, 2))
	for _, b := range bases {
		slot := at(b)
		v, err := view(map[string]any{token: map[string]any{"stateDiff": map[string]string{slot.Hex(): common.BigToHash(marker).Hex()}}})
		if err == nil && v.Cmp(marker) == 0 {
			return slot, cur, nil
		}
	}
	return common.Hash{}, nil, fmt.Errorf("no slot found")
}

// lotBalanceSlot finds the ERC-20 balance slot of `who` by probing mapping base slots with a state override.
func lotBalanceSlot(t *testing.T, ctx context.Context, rc *rpc.Client, token, who string) common.Hash {
	t.Helper()
	marker := big.NewInt(0x1234567)
	for base := int64(0); base < 12; base++ {
		slot := crypto.Keccak256Hash(encAddr(who), encWord(big.NewInt(base)))
		var out hexutil.Bytes
		msg := map[string]any{"to": token, "data": hexutil.Bytes(append(sel("balanceOf(address)"), encAddr(who)...))}
		ov := map[string]any{token: map[string]any{"stateDiff": map[string]string{slot.Hex(): common.BigToHash(marker).Hex()}}}
		if err := rc.CallContext(ctx, &out, "eth_call", msg, "latest", ov); err == nil && wordAt(out, 0).Cmp(marker) == 0 {
			return slot
		}
	}
	t.Fatalf("no balance slot for %s", token)
	return common.Hash{}
}

var _ = fmt.Sprint
