package flywheelfun

import (
	"math/big"
	"os"
	"testing"

	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func FuzzCompositeQuotePurity(f *testing.F) {
	raw, err := os.ReadFile("testdata/fork-markets.json")
	if err != nil {
		f.Fatal(err)
	}
	var markets []entity.Pool
	if err = json.Unmarshal(raw, &markets); err != nil {
		f.Fatal(err)
	}
	for i := range markets {
		for _, a := range []uint64{1, 100, 1000000000, 400000000000, 10000000000000} {
			f.Add(uint8(i), a)
		}
	}
	f.Fuzz(func(t *testing.T, index uint8, amount uint64) {
		p := markets[int(index)%len(markets)]
		s, err := NewPoolSimulator(p)
		if err != nil {
			t.Fatal(err)
		}
		params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: WETH, Amount: new(big.Int).SetUint64(amount)}, TokenOut: p.Address}
		input := new(big.Int).Set(params.TokenAmountIn.Amount)
		curve := s.Curve
		var before []string
		for _, b := range s.Bases {
			before = append(before, fingerprint(b))
		}
		first, e1 := s.CalcAmountOut(params)
		second, e2 := s.CalcAmountOut(params)
		if (e1 == nil) != (e2 == nil) || curve != s.Curve || input.Cmp(params.TokenAmountIn.Amount) != 0 {
			t.Fatal("quote mutated state or input")
		}
		for i, b := range s.Bases {
			if before[i] != fingerprint(b) {
				t.Fatal("route state mutated")
			}
		}
		if e1 != nil {
			return
		}
		if first.TokenAmountOut.Amount.Cmp(second.TokenAmountOut.Amount) != 0 || first.TokenAmountOut.Amount.Sign() <= 0 || first.Fee.Amount.Sign() < 0 || first.RemainingTokenAmountIn.Amount.Sign() < 0 || first.RemainingTokenAmountIn.Amount.Cmp(input) > 0 {
			t.Fatal("invalid or nondeterministic quote")
		}
		cloned := s.CloneState().(*PoolSimulator)
		cloned.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: first.SwapInfo})
		third, err := s.CalcAmountOut(params)
		if err != nil || third.TokenAmountOut.Amount.Cmp(first.TokenAmountOut.Amount) != 0 {
			t.Fatal("clone update changed original")
		}
		info := first.SwapInfo.(SwapInfo)
		if _, err = EncodeTradeData(info, 50, 100, 400); err != nil {
			t.Fatal(err)
		}
	})
}
