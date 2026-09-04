package ilyris

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// SwapInfo travels from CalcAmountOut to UpdateBalance so the state update does not have to
// re-derive what the quote already computed. Their framework passes it back verbatim.
//
// Every field is json:"-": this is in-process handoff, never serialised, and tagging it out
// keeps it from leaking into a route response.
type SwapInfo struct {
	NewActiveID int32     `json:"-"`
	XForY       bool      `json:"-"`
	BinsCrossed int       `json:"-"`
	Fills       []BinFill `json:"-"`
}

// CalcAmountOut prices a swap. One of the three methods the embedded base cannot supply.
//
// The bin traversal is NOT reimplemented here. It calls the in-package kernel
// (pool.go / math.go), whose oracle tests assert wei-exact parity against the
// deployed contract on generated vectors. Rewriting that math in this file
// would create a second implementation to keep in agreement with the first,
// and the failure mode of a drifting price model is mispriced routes rather
// than a crash -- silent, and paid for by whoever trades against us.
func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if err := p.blocked(); err != nil {
		return nil, err
	}

	inIdx := p.tokenIndex(params.TokenAmountIn.Token)
	outIdx := p.tokenIndex(params.TokenOut)
	if inIdx < 0 || outIdx < 0 || inIdx == outIdx {
		return nil, ErrInvalidToken
	}
	amountIn := params.TokenAmountIn.Amount
	if amountIn == nil || amountIn.Sign() <= 0 {
		return nil, ErrInsufficientLiquidity
	}

	// Index 0 is tokenX by construction of Info.Tokens, so selling index 0 is xForY.
	xForY := inIdx == 0

	sim, err := p.kernel()
	if err != nil {
		return nil, err
	}
	q, err := sim.QuoteExactIn(xForY, amountIn)
	if err != nil {
		// The kernel already distinguishes "cannot fill" from "bad input". Surfacing its
		// error rather than a zero keeps that distinction: a zero amountOut would be routed
		// as a real quote of nothing.
		return nil, err
	}
	if q.AmountOut == nil || q.AmountOut.Sign() <= 0 {
		return nil, ErrInsufficientLiquidity
	}

	crossed := binsCrossed(p.activeID, int32(q.FinalID))
	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: q.AmountOut},
		// Denominated in the INPUT token, because that is where BinPool takes it:
		// netIn = amountIn * (FEE_PRECISION - rate) / FEE_PRECISION, charged once before any
		// bin is touched rather than per bin.
		Fee:      &pool.TokenAmount{Token: params.TokenAmountIn.Token, Amount: q.FeeAmount},
		Gas:      gasFor(crossed),
		SwapInfo: SwapInfo{NewActiveID: int32(q.FinalID), XForY: xForY, BinsCrossed: crossed, Fills: q.Fills},
	}, nil
}

// UpdateBalance applies a swap the router decided to take.
func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	si, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		// RETURN, do not continue. The liquiditybookv21 template logs and carries on into a
		// nil dereference; a failed assertion here means we cannot know which direction the
		// swap went, and guessing corrupts the book for every later route in the request.
		return
	}

	inIdx := p.tokenIndex(params.TokenAmountIn.Token)
	outIdx := p.tokenIndex(params.TokenAmountOut.Token)
	if inIdx < 0 || outIdx < 0 {
		return
	}

	// NET of fee, not gross. The fee is skimmed before the input reaches any bin, so adding
	// the gross amount would inflate reserves by the fee on every single swap and drift the
	// book further from chain with each one.
	netIn := new(big.Int).Set(params.TokenAmountIn.Amount)
	if params.Fee.Amount != nil {
		netIn.Sub(netIn, params.Fee.Amount)
	}
	p.Info.Reserves[inIdx] = new(big.Int).Add(p.Info.Reserves[inIdx], netIn)
	p.Info.Reserves[outIdx] = new(big.Int).Sub(p.Info.Reserves[outIdx], params.TokenAmountOut.Amount)

	// Per-bin reserves must move with the swap. Leaving them untouched is what
	// made split/multi-hop re-quotes see the original book and overstate depth.
	p.applyFills(si.Fills)
	p.activeID = si.NewActiveID
}

func (p *PoolSimulator) applyFills(fills []BinFill) {
	for _, f := range fills {
		i := p.binIndex(int32(f.ID))
		if i < 0 {
			continue
		}
		b := p.bins[i]
		nx := new(big.Int).Set(b.ReserveX)
		ny := new(big.Int).Set(b.ReserveY)
		if f.AmountXIn != nil {
			nx.Add(nx, f.AmountXIn)
		}
		if f.AmountXOut != nil {
			nx.Sub(nx, f.AmountXOut)
		}
		if f.AmountYIn != nil {
			ny.Add(ny, f.AmountYIn)
		}
		if f.AmountYOut != nil {
			ny.Sub(ny, f.AmountYOut)
		}
		if nx.Sign() < 0 {
			nx = new(big.Int)
		}
		if ny.Sign() < 0 {
			ny = new(big.Int)
		}
		p.bins[i] = bin{ID: b.ID, ReserveX: nx, ReserveY: ny}
	}
}

func (p *PoolSimulator) binIndex(id int32) int {
	for i, b := range p.bins {
		if b.ID == id {
			return i
		}
	}
	return -1
}

// binsCrossed counts levels traversed, inclusive of the one the swap started in.
func binsCrossed(from, to int32) int {
	d := int(to - from)
	if d < 0 {
		d = -d
	}
	return d + 1
}

// gasFor mirrors BinPoolLens.estimateSwapGas: a base cost plus a linear per-extra-bin cost.
// A flat constant would flatter large swaps and penalise small ones, which is exactly the
// distortion that makes a router pick the wrong venue.
func gasFor(crossed int) int64 {
	if crossed <= 1 {
		return BaseSwapGas
	}
	return BaseSwapGas + int64(crossed-1)*PerExtraBinGas
}

// kernel builds the in-package parity-tested bin book from this adapter's snapshot.
func (p *PoolSimulator) kernel() (*binSimulator, error) {
	bins := make(map[int]BinReserves, len(p.bins))
	for _, b := range p.bins {
		bins[int(b.ID)] = BinReserves{
			ReserveX: new(big.Int).Set(b.ReserveX),
			ReserveY: new(big.Int).Set(b.ReserveY),
		}
	}
	return newBinSimulator(PoolParams{
		BinStepBps: int(p.binStepBps),
		ActiveID:   int(p.activeID),
		DecimalsX:  p.decimalsX,
		DecimalsY:  p.decimalsY,
		// The kernel takes the fee as bps and applies the contract's own formula. Passing the
		// already-resolved total rate keeps the volatility component the tracker read from
		// chain rather than re-deriving it here, which would be a second fee model.
		SwapFeeBps: int(p.totalFeeRate * bps / feePrecision),
	}, bins)
}
