package lunarbase

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// Controlled sensitivity experiment: preserve every field of one real pinned
// snapshot, inject the extra reserve delta from the reproduced 60-USDT swap,
// and compare quotes. These are not historical quotes at that swap's block.
func TestCounterfactualReserveFaultCanLeaveQuotesUnchanged(t *testing.T) {
	p := cfEntity(t)
	bad := p
	bad.Reserves = append(entity.PoolReserves(nil), p.Reserves...)
	x, _ := new(big.Int).SetString(bad.Reserves[0], 10)
	x.Sub(x, big.NewInt(81422636410922337))
	bad.Reserves[0] = x.String()
	y, _ := new(big.Int).SetString(bad.Reserves[1], 10)
	y.Add(y, new(big.Int).Mul(big.NewInt(60), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)))
	bad.Reserves[1] = y.String()
	sizes := []string{"0.000001", "0.01", "0.1", "1", "10", "50", "100", "300", "1000", "10000"}
	rows := []map[string]any{}
	up, down := 0, 0
	for side := 0; side < 2; side++ {
		for _, size := range sizes {
			amount, _ := new(big.Rat).SetString(size)
			amount.Mul(amount, new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)))
			if side == 0 {
				amount.Quo(amount, big.NewRat(7361365785, 10000000))
			}
			input := new(big.Int).Quo(amount.Num(), amount.Denom())
			goodResult, goodErr := cfCalc(cfSimulator(t, p), side, input)
			badResult, badErr := cfCalc(cfSimulator(t, bad), side, input)
			if goodErr != nil || badErr != nil {
				t.Fatalf("size %s side%d unexpected quote error %v/%v", size, side, goodErr, badErr)
			}
			delta := new(big.Int).Sub(goodResult.TokenAmountOut.Amount, badResult.TokenAmountOut.Amount)
			if delta.Sign() > 0 {
				up++
			}
			if delta.Sign() < 0 {
				down++
			}
			rows = append(rows, map[string]any{"side": []string{"sell", "buy"}[side], "sizeUsdtEquivalent": size, "input": input.String(), "correctOutput": goodResult.TokenAmountOut.Amount.String(), "corruptOutput": badResult.TokenAmountOut.Amount.String(), "correctionDeltaWei": delta.String()})
		}
	}
	if up != 0 || down != 0 {
		t.Fatalf("frozen sensitivity observation changed: %d/%d", up, down)
	}
	// At the inventory boundary, the same reserve fault changes eligibility
	// in opposite directions, despite unchanged ordinary-sized quotes.
	boundaries := []map[string]any{}
	anchor := cfSimulator(t, p).SqrtPriceX96.ToBig()
	square := new(big.Int).Mul(anchor, anchor)
	scale := new(big.Int).Lsh(big.NewInt(1), 192)
	for side := 0; side < 2; side++ {
		outIndex := 1 - side
		goodReserve, _ := new(big.Int).SetString(p.Reserves[outIndex], 10)
		badReserve, _ := new(big.Int).SetString(bad.Reserves[outIndex], 10)
		mid := new(big.Int).Add(goodReserve, badReserve)
		mid.Div(mid, big.NewInt(2))
		numerator, denominator := new(big.Int), new(big.Int)
		if side == 0 {
			numerator.Mul(mid, scale)
			denominator.Set(square)
		} else {
			numerator.Mul(mid, square)
			denominator.Set(scale)
		}
		input := new(big.Int).Div(numerator, denominator)
		input.Add(input, big.NewInt(1))
		g, ge := cfCalc(cfSimulator(t, p), side, input)
		b, be := cfCalc(cfSimulator(t, bad), side, input)
		if side == 0 && (ge == nil || be != nil) {
			t.Fatalf("sell boundary expected corrected reject and corrupt accept: %v/%v", ge, be)
		}
		if side == 1 && (ge != nil || be == nil) {
			t.Fatalf("buy boundary expected corrected accept and corrupt reject: %v/%v", ge, be)
		}
		boundaries = append(boundaries, map[string]any{"side": []string{"sell", "buy"}[side], "input": input.String(), "correct": cfQuote(g, ge), "corrupt": cfQuote(b, be)})
	}
	cfSave(t, "reserve-price-sensitivity.json", map[string]any{"kind": "controlled scenario, not historical replay quotes", "block": p.BlockNumber, "sourceSwapTx": "0x9b4a156e311f6fd290dd61673bbbac1b79cc11a8ff1f88b4bf1ec95f80fcc250", "correctReserves": p.Reserves, "corruptReserves": bad.Reserves, "rows": rows, "improved": up, "worsened": down, "unchanged": len(rows), "boundaries": boundaries})
	t.Logf("20 controlled comparisons unchanged despite reserve fault; two near-reserve boundaries change eligibility in opposite directions")
}
