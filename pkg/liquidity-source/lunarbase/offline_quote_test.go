package lunarbase

import (
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func offlineQuoteEntity(t *testing.T) entity.Pool {
	t.Helper()
	b, e := os.ReadFile(filepath.Join("testdata/quote-audit", "fresh-entity.json"))
	if e != nil {
		t.Fatal(e)
	}
	var p entity.Pool
	if e = json.Unmarshal(b, &p); e != nil {
		t.Fatal(e)
	}
	return p
}
func offlineSafeNew(p entity.Pool) (s *PoolSimulator, e error) {
	defer func() {
		if v := recover(); v != nil {
			e = fmt.Errorf("PANIC: %v", v)
		}
	}()
	return NewPoolSimulator(pool.FactoryParams{EntityPool: p, ChainID: valueobject.ChainIDBSC})
}

func TestOfflineQuoteMalformedEntity(t *testing.T) {
	cases := []struct {
		name  string
		alter func(*entity.Pool)
	}{
		{"one-token", func(p *entity.Pool) { p.Tokens = p.Tokens[:1] }},
		{"three-tokens", func(p *entity.Pool) { p.Tokens = append(p.Tokens, p.Tokens[0]) }},
		{"one-reserve", func(p *entity.Pool) { p.Reserves = p.Reserves[:1] }},
		{"three-reserves", func(p *entity.Pool) { p.Reserves = append(p.Reserves, "1") }},
		{"nil-token", func(p *entity.Pool) { p.Tokens[0] = nil }},
		{"empty-token", func(p *entity.Pool) { p.Tokens[0].Address = "" }},
		{"duplicate-token", func(p *entity.Pool) { p.Tokens[0].Address = p.Tokens[1].Address }},
		{"invalid-reserve", func(p *entity.Pool) { p.Reserves[0] = "abc" }},
		{"negative-reserve", func(p *entity.Pool) { p.Reserves[0] = "-1" }},
		{"overflow-uint112", func(p *entity.Pool) { p.Reserves[0] = new(big.Int).Lsh(big.NewInt(1), 112).String() }},
		{"overflow-uint256", func(p *entity.Pool) { p.Reserves[0] = new(big.Int).Lsh(big.NewInt(1), 256).String() }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := offlineQuoteEntity(t)
			c.alter(&p)
			_, e := offlineSafeNew(p)
			if e == nil {
				t.Fatal("malformed entity accepted")
			}
			if len(e.Error()) >= 6 && e.Error()[:6] == "PANIC:" {
				t.Fatal(e)
			}
		})
	}
}

func TestOfflineQuoteStaleWithoutFactoryCheck(t *testing.T) {
	p := offlineQuoteEntity(t)
	var extra Extra
	if e := json.Unmarshal([]byte(p.Extra), &extra); e != nil {
		t.Fatal(e)
	}
	p.BlockNumber = extra.LatestUpdateBlock + extra.BlockDelay
	s, e := NewPoolSimulator(pool.FactoryParams{EntityPool: p, ChainID: valueobject.ChainIDBSC, Opts: pool.FactoryOpts{StaleCheck: false}})
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: p.Tokens[1].Address, Amount: big.NewInt(1000000000000000000)}, TokenOut: p.Tokens[0].Address})
	if e != ErrStalePool {
		t.Fatalf("got %v, want stale rejection at boundary even with factory option off", e)
	}
}

func TestOfflineQuoteUpdateConsumesOnlySwapInfo(t *testing.T) {
	s := offlineQuoteSimulator(t)
	p := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: s.Info.Tokens[1], Amount: big.NewInt(1000000000000000000)}, TokenOut: s.Info.Tokens[0]}
	r, e := s.CalcAmountOut(p)
	if e != nil {
		t.Fatal(e)
	}
	baseline, perturbed := s.CloneState().(*PoolSimulator), s.CloneState().(*PoolSimulator)
	baseline.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: p.TokenAmountIn, TokenAmountOut: *r.TokenAmountOut, Fee: *r.Fee, SwapInfo: r.SwapInfo})
	perturbed.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: pool.TokenAmount{Token: p.TokenAmountIn.Token, Amount: big.NewInt(1)}, TokenAmountOut: pool.TokenAmount{Token: p.TokenOut, Amount: big.NewInt(1)}, Fee: pool.TokenAmount{Token: p.TokenOut, Amount: big.NewInt(999)}, SwapInfo: r.SwapInfo})
	for i := range baseline.reserves {
		if !baseline.reserves[i].Eq(perturbed.reserves[i]) {
			t.Fatal("UpdateBalance trusted mutable amounts instead of quoted SwapInfo")
		}
		if baseline.GetReserves()[i].Cmp(baseline.reserves[i].ToBig()) != 0 {
			t.Fatal("public reserves differ from quote state")
		}
	}
	noInfo := s.CloneState().(*PoolSimulator)
	noInfo.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: p.TokenAmountIn, TokenAmountOut: *r.TokenAmountOut, Fee: *r.Fee, SwapInfo: nil})
	for i := range s.reserves {
		if !s.reserves[i].Eq(noInfo.reserves[i]) {
			t.Fatal("UpdateBalance accepted missing SwapInfo")
		}
	}
}

func offlineQuoteSimulator(t *testing.T) *PoolSimulator {
	t.Helper()
	s, e := NewPoolSimulator(pool.FactoryParams{EntityPool: offlineQuoteEntity(t), ChainID: valueobject.ChainIDBSC})
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func offlineSafeCalc(s *PoolSimulator, p pool.CalcAmountOutParams) (r *pool.CalcAmountOutResult, e error) {
	defer func() {
		if v := recover(); v != nil {
			e = fmt.Errorf("PANIC: %v", v)
		}
	}()
	return s.CalcAmountOut(p)
}
func TestOfflineQuoteInputValidation(t *testing.T) {
	s := offlineQuoteSimulator(t)
	one := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	cases := []struct {
		name string
		p    pool.CalcAmountOutParams
	}{
		{"same-token", pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: s.Info.Tokens[1], Amount: one}, TokenOut: s.Info.Tokens[1]}},
		{"nil-amount", pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: s.Info.Tokens[1], Amount: nil}, TokenOut: s.Info.Tokens[0]}},
		{"negative-amount", pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: s.Info.Tokens[1], Amount: big.NewInt(-1)}, TokenOut: s.Info.Tokens[0]}},
		{"overflow-amount", pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: s.Info.Tokens[1], Amount: new(big.Int).Lsh(big.NewInt(1), 256)}, TokenOut: s.Info.Tokens[0]}},
		{"zero-amount", pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: s.Info.Tokens[1], Amount: big.NewInt(0)}, TokenOut: s.Info.Tokens[0]}},
	}
	out := []map[string]any{}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, e := offlineSafeCalc(s, c.p)
			row := map[string]any{"case": c.name, "error": fmt.Sprint(e)}
			if r != nil {
				row["amountOut"] = r.TokenAmountOut.Amount.String()
			}
			out = append(out, row)
			if e == nil {
				t.Error("invalid input was accepted")
			}
			if e != nil && len(e.Error()) >= 6 && e.Error()[:6] == "PANIC:" {
				t.Error(e)
			}
		})
	}
}

func TestOfflineQuoteCloneAndPurity(t *testing.T) {
	s := offlineQuoteSimulator(t)
	before, _ := json.Marshal(s)
	params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: s.Info.Tokens[1], Amount: new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil)}, TokenOut: s.Info.Tokens[0]}
	r, e := s.CalcAmountOut(params)
	if e != nil {
		t.Fatal(e)
	}
	again, _ := json.Marshal(s)
	if string(again) != string(before) {
		t.Fatal("CalcAmountOut mutated simulator")
	}
	clone := s.CloneState().(*PoolSimulator)
	clone.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: params.TokenAmountIn, TokenAmountOut: *r.TokenAmountOut, Fee: *r.Fee, SwapInfo: r.SwapInfo})
	after, _ := json.Marshal(s)
	if string(after) != string(before) {
		t.Fatal("clone update mutated original public state")
	}
	for i := range s.reserves {
		if s.reserves[i].Eq(clone.reserves[i]) {
			t.Fatalf("clone reserve%d did not update", i)
		}
	}
	r2, e := s.CalcAmountOut(params)
	if e != nil || r2.TokenAmountOut.Amount.Cmp(r.TokenAmountOut.Amount) != 0 {
		t.Fatal("repeat quote changed after clone update")
	}
}

func TestOfflineQuoteStatefulDifferential(t *testing.T) {
	b, e := os.ReadFile(filepath.Join("testdata/quote-audit", "stateful-executions.json"))
	if e != nil {
		t.Fatal(e)
	}
	var cases []struct {
		Scenario string `json:"scenario"`
		Block    uint64 `json:"block"`
		Decoded  *struct {
			Initial struct {
				Reserves []string `json:"reserves"`
				State    []string `json:"state"`
			} `json:"initial"`
			Steps []struct {
				Side          int      `json:"side"`
				Input         string   `json:"input"`
				QuoteBefore   []string `json:"quoteBefore"`
				SwapOut       string   `json:"swapOut"`
				ReservesAfter []string `json:"reservesAfter"`
				StateAfter    []string `json:"stateAfter"`
				QuoteAfter    []string `json:"quoteAfterSameAmount"`
			} `json:"steps"`
		} `json:"decoded"`
	}
	if e = json.Unmarshal(b, &cases); e != nil {
		t.Fatal(e)
	}
	counts := map[string]int{}
	for _, c := range cases {
		if c.Decoded == nil {
			t.Fatalf("missing execution result %s", c.Scenario)
		}
		s := offlineQuoteSimulator(t)
		if s.Info.BlockNumber != c.Block {
			t.Fatal("snapshot and execution blocks differ")
		}
		for i, step := range c.Decoded.Steps {
			a, _ := new(big.Int).SetString(step.Input, 10)
			p := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: s.Info.Tokens[step.Side], Amount: a}, TokenOut: s.Info.Tokens[1-step.Side]}
			r, err := offlineSafeCalc(s, p)
			if err != nil {
				t.Fatal(err)
			}
			beforeOutMatch := r.TokenAmountOut.Amount.String() == step.QuoteBefore[0]
			beforeFeeMatch := r.Fee.Amount.String() == step.QuoteBefore[2]
			s.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: p.TokenAmountIn, TokenAmountOut: *r.TokenAmountOut, Fee: *r.Fee, SwapInfo: r.SwapInfo})
			reservesMatch := s.reserves[0].Dec() == step.ReservesAfter[0] && s.reserves[1].Dec() == step.ReservesAfter[1]
			feesMatch := fmt.Sprint(s.FeeAskX24) == step.StateAfter[1] && fmt.Sprint(s.FeeBidX24) == step.StateAfter[2]
			next, err := offlineSafeCalc(s, p)
			nextOut, nextFee := "0", "0"
			if err == nil {
				nextOut = next.TokenAmountOut.Amount.String()
				nextFee = next.Fee.Amount.String()
			}
			afterMatch := nextOut == step.QuoteAfter[0] && (err != nil || nextFee == step.QuoteAfter[2])
			row := map[string]any{"scenario": c.Scenario, "step": i, "input": step.Input, "side": step.Side, "beforeOutMatch": beforeOutMatch, "beforeFeeMatch": beforeFeeMatch, "reservesMatch": reservesMatch, "feesMatch": feesMatch, "afterQuoteMatch": afterMatch, "simReserves": []string{s.reserves[0].Dec(), s.reserves[1].Dec()}, "chainReserves": step.ReservesAfter, "simBeforeOut": r.TokenAmountOut.Amount.String(), "simBeforeFee": r.Fee.Amount.String(), "chainBeforeQuote": step.QuoteBefore, "simAfterOut": nextOut, "simAfterFee": nextFee, "chainAfterQuote": step.QuoteAfter}
			counts["steps"]++
			for _, key := range []string{"beforeOutMatch", "beforeFeeMatch", "reservesMatch", "feesMatch", "afterQuoteMatch"} {
				if row[key].(bool) {
					counts[key]++
				}
			}
		}
	}
	t.Logf("counts=%v", counts)
	for _, key := range []string{"beforeOutMatch", "beforeFeeMatch", "reservesMatch", "feesMatch", "afterQuoteMatch"} {
		if counts[key] != counts["steps"] {
			t.Errorf("%s: %d/%d", key, counts[key], counts["steps"])
		}
	}
}

func TestOfflineQuoteSavedRPC(t *testing.T) {
	b, e := os.ReadFile("testdata/quote-audit/fresh-differential.json")
	if e != nil {
		t.Fatal(e)
	}
	var fixture struct {
		Block uint64 `json:"block"`
		Rows  []struct {
			Side       int    `json:"side"`
			Input      string `json:"input"`
			ChainOut   string `json:"chainOut"`
			ChainFee   string `json:"chainFee"`
			ChainError string `json:"chainError"`
		} `json:"rows"`
	}
	if e = json.Unmarshal(b, &fixture); e != nil {
		t.Fatal(e)
	}
	s := offlineQuoteSimulator(t)
	if fixture.Block != s.Info.BlockNumber || len(fixture.Rows) != 512 {
		t.Fatal("wrong saved snapshot")
	}
	exact, rejected := 0, 0
	for i, r := range fixture.Rows {
		amount, ok := new(big.Int).SetString(r.Input, 10)
		if !ok {
			t.Fatal(r.Input)
		}
		result, err := offlineSafeCalc(s, pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: s.Info.Tokens[r.Side], Amount: amount}, TokenOut: s.Info.Tokens[1-r.Side]})
		if r.ChainError != "" {
			if !strings.Contains(r.ChainError, "execution reverted") || err == nil {
				t.Errorf("row %d: rejection differs", i)
			} else {
				rejected++
			}
			continue
		}
		out, fee := "0", "0"
		if err == nil {
			out = result.TokenAmountOut.Amount.String()
			fee = result.Fee.Amount.String()
		}
		if out != r.ChainOut || (err == nil && fee != r.ChainFee) {
			t.Errorf("row %d: got %s/%s, chain %s/%s", i, out, fee, r.ChainOut, r.ChainFee)
		} else {
			exact++
		}
	}
	if exact != 504 || rejected != 8 {
		t.Fatalf("exact=%d rejected=%d", exact, rejected)
	}
	t.Logf("504 exact outputs/fees; 8 shared rejections at block %d", fixture.Block)
}

func TestOfflineQuoteParameterGrid(t *testing.T) {
	b, e := os.ReadFile("testdata/quote-audit/parameter-grid.json")
	if e != nil {
		t.Fatal(e)
	}
	var f struct {
		Block uint64 `json:"block"`
		Rows  []struct {
			Side  int      `json:"side"`
			Input string   `json:"input"`
			Max   uint32   `json:"max"`
			Fee   uint32   `json:"fee"`
			Words []string `json:"words"`
		} `json:"rows"`
	}
	if e = json.Unmarshal(b, &f); e != nil {
		t.Fatal(e)
	}
	base := offlineQuoteSimulator(t)
	if f.Block != base.Info.BlockNumber || len(f.Rows) != 160 {
		t.Fatal("wrong grid snapshot")
	}
	for i, r := range f.Rows {
		p := &PoolParams{SqrtPriceX96: base.SqrtPriceX96, ReserveX: base.reserves[0], ReserveY: base.reserves[1], FeeAskX24: r.Fee, FeeBidX24: r.Fee, MaxPunishmentX24: r.Max}
		amount := uint256.MustFromDecimal(r.Input)
		var q *QuoteResult
		if r.Side == 0 {
			q = quoteXToY(p, amount)
		} else {
			q = quoteYToX(p, amount)
		}
		got := []string{q.AmountOut.Dec(), q.SqrtPriceNext.Dec(), q.Fee.Dec()}
		if fmt.Sprint(got) != fmt.Sprint(r.Words) {
			t.Errorf("row %d max=%d fee=%d side=%d amount=%s got%v want%v", i, r.Max, r.Fee, r.Side, r.Input, got, r.Words)
		}
	}
	t.Log("160 deployed-code counterfactual quotes: max punishment and fee boundaries")
}

func TestOfflineQuoteStaleRuntime(t *testing.T) {
	b, e := os.ReadFile("testdata/quote-audit/stale-runtime.json")
	if e != nil {
		t.Fatal(e)
	}
	var f []struct {
		Offset        int      `json:"offset"`
		SnapshotBlock uint64   `json:"snapshotBlock"`
		OverrideBlock uint64   `json:"overrideBlock"`
		Success       bool     `json:"success"`
		Words         []string `json:"words"`
	}
	if e = json.Unmarshal(b, &f); e != nil {
		t.Fatal(e)
	}
	if len(f) != 3 {
		t.Fatal("wrong boundary fixture")
	}
	for _, r := range f {
		s := offlineQuoteSimulator(t)
		if s.Info.BlockNumber != r.SnapshotBlock || !r.Success {
			t.Fatal("wrong boundary snapshot")
		}
		s.Info.BlockNumber = r.OverrideBlock
		result, err := s.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: s.Info.Tokens[1], Amount: big.NewInt(1000000000000000000)}, TokenOut: s.Info.Tokens[0]})
		out := "0"
		if err == nil {
			out = result.TokenAmountOut.Amount.String()
		}
		if out != r.Words[0] {
			t.Errorf("offset%d output %s want%s", r.Offset, out, r.Words[0])
		}
		if r.Offset >= 0 && err != ErrStalePool {
			t.Errorf("offset%d must reject stale, got%v", r.Offset, err)
		}
	}
}

func TestOfflineQuoteMetadata(t *testing.T) {
	s := offlineQuoteSimulator(t)
	s.BlockHash = "0x1111111111111111111111111111111111111111111111111111111111111111"
	m := s.GetMetaInfo(s.Info.Tokens[0], s.Info.Tokens[1]).(PoolMeta)
	if m.BlockHash != s.BlockHash || m.BlockNumber != s.Info.BlockNumber {
		t.Fatal("snapshot metadata lost")
	}
	if !s.HasNative || !s.SwapReceiveNativeIn(s.Info.Tokens[0], s.Info.Tokens[1], valueobject.ChainIDBSC) || !s.SwapReturnNativeOut(s.Info.Tokens[1], s.Info.Tokens[0], valueobject.ChainIDBSC) {
		t.Fatal("native mapping lost")
	}
}
