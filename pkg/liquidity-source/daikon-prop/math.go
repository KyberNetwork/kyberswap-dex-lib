package daikonprop

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const (
	bps         = 10_000
	feeDenomPpm = 1_000_000
)

var (
	wad     = big256.BONE
	uBps    = big256.UBasisPoint
	uFeeDen = uint256.NewInt(feeDenomPpm)
)

// tierPrice is the spread-adjusted price. A taker buying base pays up (ceil), selling receives down.
func tierPrice(res, price *uint256.Int, spreadBps uint64, buy bool) *uint256.Int {
	var adj uint256.Int
	if buy {
		adj.SetUint64(bps + spreadBps)
		return big256.MulDivUp(res, price, &adj, uBps)
	}
	if spreadBps > bps {
		return res.Clear()
	}
	adj.SetUint64(bps - spreadBps)
	return big256.MulDivDown(res, price, &adj, uBps)
}

// quoteExactIn ports the pair's exact-input pricing from the Daikon docs. It writes the net
// amountOut and the quote-unit capacity consumed per tier into info. The pair is unverified, so
// the rounding is pinned by TestQuoteMatchesOnChain.
func (e *Extra) quoteExactIn(amountIn *uint256.Int, buy bool, feePpm uint64, amountOut *uint256.Int,
	info *SwapInfo) error {
	feePpm = min(feePpm, maxFeePpm)
	// Decimals are validated <= 18 in NewPoolSimulator. TenPow(18-d) takes native units to WAD.
	qs, bs := big256.TenPow(18-e.QuoteDecimals), big256.TenPow(18-e.BaseDecimals)
	var ppm uint256.Int
	ppm.SetUint64(feePpm)
	info.BuyBase = buy
	info.C1Used, info.C2Used = new(uint256.Int), new(uint256.Int)

	var p1, p2, t, tier1, tier2 uint256.Int
	if buy {
		// Step 1: fee off the input, then split across the remaining buy capacity.
		var feeIn, pairIn uint256.Int
		big256.MulDivUp(&feeIn, amountIn, &ppm, uFeeDen)
		pairIn.Sub(amountIn, &feeIn)
		info.C1Used.Set(e.BuyC1)
		if pairIn.Lt(info.C1Used) {
			info.C1Used.Set(&pairIn)
		}
		info.C2Used.Sub(&pairIn, info.C1Used)
		if info.C2Used.Gt(e.BuyC2) {
			return ErrCapacity
		}
		// Step 3: price each tier and floor once at the end.
		tierPrice(&p1, e.Price, e.Spread1Bps, true)
		tierPrice(&p2, e.Price, e.Spread2Bps, true)
		if p1.IsZero() || p2.IsZero() {
			return ErrNotTrading
		}
		// q <= capacity, a published 96-bit figure, so q*qs and q*qs*WAD fit 256 bits.
		tier1.Mul(info.C1Used, qs)
		big256.MulDivDown(&tier1, &tier1, wad, &p1)
		tier2.Mul(info.C2Used, qs)
		big256.MulDivDown(&tier2, &tier2, wad, &p2)
		t.Add(&tier1, &tier2)
		amountOut.Div(&t, bs)
		return nil
	}

	// Sell: tier capacities in base at the mid price.
	var cap1, cap2, inWad, t1, t2 uint256.Int
	if e.Price.IsZero() {
		return ErrNotTrading
	}
	t.Mul(e.SellC1, qs)
	big256.MulDivDown(&cap1, &t, wad, e.Price)
	t.Mul(e.SellC2, qs)
	big256.MulDivDown(&cap2, &t, wad, e.Price)
	if _, overflow := inWad.MulOverflow(amountIn, bs); overflow {
		return ErrCapacity
	}
	t.Add(&cap1, &cap2)
	if inWad.Gt(&t) {
		return ErrCapacity
	}
	t1.Set(&cap1)
	if inWad.Lt(&t1) {
		t1.Set(&inWad)
	}
	t2.Sub(&inWad, &t1)

	tierPrice(&p1, e.Price, e.Spread1Bps, false)
	tierPrice(&p2, e.Price, e.Spread2Bps, false)
	// t <= cap, so t*price fits 256 bits.
	big256.MulDivDown(&tier1, &t1, &p1, wad)
	big256.MulDivDown(&tier2, &t2, &p2, wad)
	t.Add(&tier1, &tier2)
	t.Div(&t, qs) // pairOut
	var feeOut uint256.Int
	big256.MulDivUp(&feeOut, &t, &ppm, uFeeDen)
	amountOut.Sub(&t, &feeOut)

	// Capacity consumed, in quote units at the mid price, rounded up (checked on a fork).
	consumed(info.C1Used, &t1, e.Price, qs)
	consumed(info.C2Used, &t2, e.Price, qs)
	return nil
}

func consumed(res, baseWad, price, qs *uint256.Int) {
	if baseWad.IsZero() {
		res.Clear()
		return
	}
	big256.MulDivUp(res, baseWad, price, wad)
	res.Set(big256.DivUp(res, qs))
}
