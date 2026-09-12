package lunarbase

import (
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var cfRoot = filepath.Join("testdata", "audit")

func cfRead(t *testing.T, name string, v any) {
	t.Helper()
	b, e := os.ReadFile(filepath.Join(cfRoot, name))
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(b, v); e != nil {
		t.Fatal(e)
	}
}
func cfSave(t *testing.T, name string, v any) {
	t.Helper()
	label := os.Getenv("KYBER_AUDIT_VARIANT")
	if label == "" || os.Getenv("LUNARBASE_AUDIT_RESULTS") == "" {
		return
	}
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(os.Getenv("LUNARBASE_AUDIT_RESULTS"), label+"-"+name), b, 0644); e != nil {
		t.Fatal(e)
	}
}
func cfEntity(t *testing.T) entity.Pool {
	t.Helper()
	var p entity.Pool
	cfRead(t, "live-entity.json", &p)
	return p
}
func cfSimulator(t *testing.T, p entity.Pool) *PoolSimulator {
	t.Helper()
	s, e := NewPoolSimulator(pool.FactoryParams{EntityPool: p, ChainID: valueobject.ChainIDBSC, Opts: pool.FactoryOpts{StaleCheck: true}})
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func cfCalc(s *PoolSimulator, side int, in *big.Int) (*pool.CalcAmountOutResult, error) {
	return s.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: s.GetTokens()[side], Amount: in}, TokenOut: s.GetTokens()[1-side]})
}
func cfFingerprint(s *PoolSimulator) string {
	b, _ := json.Marshal(struct {
		Extra    *Extra
		Reserves []string
		Meta     any
	}{s.Extra, []string{s.reserves[0].Dec(), s.reserves[1].Dec()}, s.GetMetaInfo(s.GetTokens()[0], s.GetTokens()[1])})
	return string(b)
}
func cfQuote(r *pool.CalcAmountOutResult, e error) string {
	if e != nil {
		return "error:" + e.Error()
	}
	b, _ := json.Marshal(struct {
		Out, Fee string
		Gas      int64
		Info     SwapInfo
	}{r.TokenAmountOut.Amount.String(), r.Fee.Amount.String(), r.Gas, r.SwapInfo.(SwapInfo)})
	return string(b)
}

func TestCounterfactualSavedOnChainQuotes(t *testing.T) {
	p := cfEntity(t)
	var fixture struct {
		Block uint64                                                     `json:"block"`
		Rows  []struct{ Side, Input, Caller, ChainOut, ChainFee string } `json:"rows"`
	}
	cfRead(t, "pinned-go-quotes.json", &fixture)
	if p.BlockNumber != fixture.Block || len(fixture.Rows) != 116 {
		t.Fatal("wrong frozen fixture")
	}
	results := []map[string]any{}
	positive := 0
	for i, row := range fixture.Rows {
		side := 0
		if row.Side == "buy" {
			side = 1
		}
		s := cfSimulator(t, p)
		in, ok := new(big.Int).SetString(row.Input, 10)
		if !ok {
			t.Fatal(row.Input)
		}
		before := cfFingerprint(s)
		r, e := cfCalc(s, side, in)
		out, fee := "0", "0"
		if e == nil {
			out = r.TokenAmountOut.Amount.String()
			fee = r.Fee.Amount.String()
			positive++
		}
		if out != row.ChainOut || (out != "0" && fee != row.ChainFee) {
			t.Errorf("row %d caller %s output/fee got %s/%s want %s/%s", i, row.Caller, out, fee, row.ChainOut, row.ChainFee)
		}
		if before != cfFingerprint(s) {
			t.Errorf("quote mutated state row %d", i)
		}
		results = append(results, map[string]any{"index": i, "caller": row.Caller, "input": row.Input, "output": out, "fee": fee, "match": out == row.ChainOut && (out == "0" || fee == row.ChainFee)})
	}
	cfSave(t, "frozen-quotes.json", map[string]any{"block": p.BlockNumber, "rows": results, "positiveQuotes": positive})
	t.Logf("116 frozen on-chain output comparisons; %d positive fee comparisons; 116 purity checks", positive)
}

func TestCounterfactualQuotePurityCloneAndSequence(t *testing.T) {
	random := rand.New(rand.NewSource(20260912))
	base := cfEntity(t)
	rows := []map[string]any{}
	success, errors := 0, 0
	for scenario := 0; scenario < 120; scenario++ {
		p := base
		var extra Extra
		if e := json.Unmarshal([]byte(p.Extra), &extra); e != nil {
			t.Fatal(e)
		}
		extra.FeeAskX24 = uint32(random.Intn(20000))
		extra.FeeBidX24 = uint32(random.Intn(20000))
		extra.MaxPunishmentX24 = uint32(random.Intn(150001))
		// This is an explicitly constructed synthetic model, not an ambiguous
		// legacy cache entry. The marker is test metadata, not a chain hash.
		extra.BlockHash = "synthetic-property-snapshot"
		if scenario%3 == 0 {
			extra.ConcentrationK = 5000
			extra.MaxPunishmentX24 = 0
		}
		if scenario%3 == 1 {
			extra.MaxPunishmentX24 = 0
		}
		b, _ := json.Marshal(extra)
		p.Extra = string(b)
		p.Reserves = append(entity.PoolReserves(nil), p.Reserves...)
		for j := range p.Reserves {
			v, _ := new(big.Int).SetString(p.Reserves[j], 10)
			v.Mul(v, big.NewInt(int64(1+random.Intn(20))))
			v.Div(v, big.NewInt(10))
			p.Reserves[j] = v.String()
		}
		s := cfSimulator(t, p)
		for step := 0; step < 30; step++ {
			side := random.Intn(2)
			in := new(big.Int).Set(s.reserves[side].ToBig())
			in.Div(in, big.NewInt(int64(20+random.Intn(5000))))
			if step%10 == 0 {
				in.SetInt64(int64(step / 10))
			}
			inputBefore := in.String()
			before := cfFingerprint(s)
			a, ea := cfCalc(s, side, in)
			qa := cfQuote(a, ea)
			b, eb := cfCalc(s, side, in)
			if qa != cfQuote(b, eb) || before != cfFingerprint(s) || in.String() != inputBefore {
				t.Fatalf("purity/determinism failed %d/%d", scenario, step)
			}
			clone := s.CloneState().(*PoolSimulator)
			c, ec := cfCalc(clone, side, in)
			if qa != cfQuote(c, ec) {
				t.Fatalf("clone quote differs %d/%d", scenario, step)
			}
			if ea == nil {
				success++
				if a.TokenAmountOut.Amount.Sign() <= 0 || a.TokenAmountOut.Amount.Cmp(s.reserves[1-side].ToBig()) > 0 {
					t.Fatalf("output outside reserve %d/%d", scenario, step)
				}
				params := pool.UpdateBalanceParams{TokenAmountIn: pool.TokenAmount{Token: s.GetTokens()[side], Amount: in}, TokenAmountOut: *a.TokenAmountOut, Fee: *a.Fee, SwapInfo: a.SwapInfo}
				clone.UpdateBalance(params)
				if before != cfFingerprint(s) {
					t.Fatalf("clone update leaked %d/%d", scenario, step)
				}
				s.UpdateBalance(params)
				if cfFingerprint(s) != cfFingerprint(clone) {
					t.Fatalf("same sequence differs %d/%d", scenario, step)
				}
			} else {
				errors++
			}
			rows = append(rows, map[string]any{"scenario": scenario, "step": step, "model": fmt.Sprint(extra.ConcentrationK), "side": side, "input": in.String(), "quote": qa, "post": cfFingerprint(s)})
		}
	}
	if success == 0 || errors == 0 {
		t.Fatal("expected mixed executable and rejected cases")
	}
	cfSave(t, "property-quotes.json", map[string]any{"seed": 20260912, "scenarios": 120, "quotes": len(rows), "successful": success, "rejected": errors, "rows": rows})
	t.Logf("120 scenarios, %d quote/clone/sequence cases: %d success, %d rejected", len(rows), success, errors)
}
