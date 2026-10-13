package btrprop

import (
	"math/big"
	"slices"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// PoolSimulator prices a BTR AIMM leg from the sampled ladder the tracker built.
// It performs no RPC: dex-lib simulators run in the route-finder hot path
// without an RPC client. The ladder is the cached on-chain quote.
type PoolSimulator struct {
	pool.Pool
	extra Extra
	// ladders[from*n+to] points into extra.Ladders; nil when the pair has no ladder.
	ladders []*Ladder
	// inUsed[i] and outTaken[j] are the totals UpdateBalance has moved in and out
	// of each token, over every pair. The core prices from per-token inventory, so
	// a route that splits one swap across pairs sharing a token (USDC>WMON plus
	// USDC>cbBTC>WMON) must not restart each leg at the first sample. Nil until first use.
	inUsed, outTaken []uint256.Int
	gated            bool
}

var _ = pool.RegisterFactory(DexType, NewPoolSimulator)

func NewPoolSimulator(params pool.FactoryParams) (*PoolSimulator, error) {
	ep := params.EntityPool
	n := len(ep.Tokens)
	if n < 2 {
		return nil, ErrBTRBadState
	}
	if params.Opts.StaleCheck && stale(ep.Timestamp) {
		return nil, ErrBTRBadState
	}

	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}
	if len(extra.Flags) != n {
		return nil, ErrBTRBadState
	}

	ladders := make([]*Ladder, n*n)
	for k := range extra.Ladders {
		l := &extra.Ladders[k]
		if l.From < 0 || l.From >= n || l.To < 0 || l.To >= n || l.From == l.To || !validPoints(l.Points) {
			continue
		}
		ladders[l.From*n+l.To] = l
	}

	reserves := make([]*big.Int, n)
	for k := range reserves {
		if k < len(ep.Reserves) {
			reserves[k] = bignumber.NewBig10(ep.Reserves[k])
		}
		if reserves[k] == nil {
			reserves[k] = new(big.Int)
		}
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     ep.Address,
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      lo.Map(ep.Tokens, func(t *entity.PoolToken, _ int) string { return t.Address }),
			Reserves:    reserves,
			BlockNumber: ep.BlockNumber,
		}},
		extra:   extra,
		ladders: ladders,
		gated:   lo.SomeBy(extra.Flags, func(f uint16) bool { return gate(f, extra.LaneOK) != nil }),
	}, nil
}

// validPoints requires a non-empty ladder with In > 0 and strictly increasing
// In and Out. The tracker guarantees this; a violation means corrupt state.
func validPoints(pts []Point) bool {
	if len(pts) == 0 || pts[0].In.IsZero() || pts[0].Out.IsZero() {
		return false
	}
	for k := 1; k < len(pts); k++ {
		if !pts[k].In.Gt(&pts[k-1].In) || !pts[k].Out.Gt(&pts[k-1].Out) {
			return false
		}
	}
	return true
}

// gate mirrors swap_qe's checks: halted or swap-disabled legs revert, and a
// SWAP_GATED leg reverts unless the executor holds lane 0x400 (laneOK).
func gate(flags uint16, laneOK bool) error {
	if flags&HaltMask != 0 || flags&SwapEnabledBit == 0 {
		return ErrBTRGated
	}
	if flags&SwapGatedBit != 0 && !laneOK {
		return ErrBTRGatedLane
	}
	return nil
}

// interpolate writes the ladder output for x into res: piecewise linear between
// samples, ErrBTRAmountTooLarge above the last. Below the first sample it takes
// the lower of two lines: the first segment extended backwards (zero once it
// crosses the axis) and the proportional line from the origin. The live curve is
// concave, so the extension sits above it by up to ~145 bps at dust, while the
// chord from the origin sits below it.
func interpolate(pts []Point, x, res *uint256.Int) error {
	if x.Gt(&pts[len(pts)-1].In) {
		return ErrBTRAmountTooLarge
	}
	if !x.Gt(&pts[0].In) {
		big256.MulDivDown(res, &pts[0].Out, x, &pts[0].In)
		if len(pts) == 1 {
			return nil
		}
		// out = p0.Out - (p0.In-x) * (p1.Out-p0.Out) / (p1.In-p0.In), rounded down
		var dy, dx, dn, ext uint256.Int
		big256.MulDivUp(&ext, dx.Sub(&pts[0].In, x), dy.Sub(&pts[1].Out, &pts[0].Out), dn.Sub(&pts[1].In, &pts[0].In))
		if ext.Gt(&pts[0].Out) {
			res.Clear()
			return nil
		}
		if ext.Sub(&pts[0].Out, &ext); ext.Lt(res) {
			res.Set(&ext)
		}
		return nil
	}
	for k := 1; k < len(pts); k++ {
		hi := &pts[k]
		if x.Gt(&hi.In) {
			continue
		}
		// out = lo.Out + (hi.Out-lo.Out) * (x-lo.In) / (hi.In-lo.In)
		lo := &pts[k-1]
		var dy, dx, dn uint256.Int
		big256.MulDivDown(res, dy.Sub(&hi.Out, &lo.Out), dx.Sub(x, &lo.In), dn.Sub(&hi.In, &lo.In))
		res.Add(res, &lo.Out)
		return nil
	}
	return ErrBTRAmountTooLarge
}

// startPoint writes into res how far along the pair's curve earlier swaps have
// already pushed: the larger of the input already sold of tokenIn and the input
// that would have yielded the output already taken of tokenOut.
func (s *PoolSimulator) startPoint(pts []Point, i, j int, res *uint256.Int) {
	if s.inUsed == nil {
		res.Clear()
		return
	}
	res.Set(&s.inUsed[i])
	if !s.outTaken[j].IsZero() {
		var x uint256.Int
		if invert(pts, &s.outTaken[j], &x); x.Gt(res) {
			res.Set(&x)
		}
	}
}

// invert writes into res the input that interpolate maps to output y, rounded
// up (the worse price). At or above the last sample it returns the last input.
func invert(pts []Point, y, res *uint256.Int) {
	if !y.Lt(&pts[len(pts)-1].Out) {
		res.Set(&pts[len(pts)-1].In)
		return
	}
	k := 0
	for y.Gt(&pts[k].Out) {
		k++
	}
	var dy, dx, dn uint256.Int
	switch {
	case k > 0: // x = lo.In + (y-lo.Out) * (hi.In-lo.In) / (hi.Out-lo.Out)
		lo, hi := &pts[k-1], &pts[k]
		big256.MulDivUp(res, dy.Sub(y, &lo.Out), dx.Sub(&hi.In, &lo.In), dn.Sub(&hi.Out, &lo.Out))
		res.Add(res, &lo.In)
	case len(pts) == 1:
		big256.MulDivUp(res, y, &pts[0].In, &pts[0].Out)
	default:
		// Below the first sample interpolate takes the lower of the extended first
		// segment (floored at 0) and the line from the origin, so the input is the
		// larger of the two inverses.
		big256.MulDivDown(res, dy.Sub(&pts[0].Out, y), dx.Sub(&pts[1].In, &pts[0].In), dn.Sub(&pts[1].Out, &pts[0].Out))
		if res.Gt(&pts[0].In) {
			res.Clear()
		} else {
			res.Sub(&pts[0].In, res)
		}
		big256.MulDivUp(&dy, y, &pts[0].In, &pts[0].Out)
		if dy.Gt(res) {
			res.Set(&dy)
		}
	}
}

// CalcAmountOut returns the BTR output for an exact-in amount, from the sampled
// ladder. It refuses a leg the executor cannot swap: halted, disabled, or
// SWAP_GATED without the executor's lane.
func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn, tokenOut := params.TokenAmountIn.Token, params.TokenOut
	i, j := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenOut)
	if i < 0 || j < 0 || i == j {
		return nil, ErrBTRBadState
	}
	var amountIn uint256.Int
	if amt := params.TokenAmountIn.Amount; amt == nil || amt.Sign() <= 0 || amountIn.SetFromBig(amt) {
		return nil, ErrBTRInvalidAmount
	}
	if err := gate(s.extra.Flags[i], s.extra.LaneOK); err != nil {
		return nil, err
	}
	if err := gate(s.extra.Flags[j], s.extra.LaneOK); err != nil {
		return nil, err
	}

	n := len(s.Info.Tokens)
	ladder := s.ladders[i*n+j]
	if ladder == nil {
		return nil, ErrBTRNoRoute
	}

	// Output for this swap = f(x0+in) - f(x0), where x0 is the point on the
	// pair's curve already consumed by earlier swaps.
	var amountOut, x0 uint256.Int
	s.startPoint(ladder.Points, i, j, &x0)
	if x0.IsZero() {
		if err := interpolate(ladder.Points, &amountIn, &amountOut); err != nil {
			return nil, err
		}
	} else {
		var base, x uint256.Int
		if _, overflow := x.AddOverflow(&amountIn, &x0); overflow {
			return nil, ErrBTRAmountTooLarge
		}
		if err := interpolate(ladder.Points, &x, &amountOut); err != nil {
			return nil, err
		}
		if err := interpolate(ladder.Points, &x0, &base); err != nil {
			return nil, err
		}
		amountOut.Sub(&amountOut, &base)
	}
	if amountOut.IsZero() {
		return nil, ErrBTRNoFill
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: tokenIn, Amount: bignumber.ZeroBI},
		Gas:            DefaultGas,
	}, nil
}

// CloneState deep-copies inUsed and outTaken, the slices UpdateBalance writes in
// place. Ladders and flags are immutable.
func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.inUsed = slices.Clone(s.inUsed)
	cloned.outTaken = slices.Clone(s.outTaken)
	return &cloned
}

// UpdateBalance records the swapped amounts per token so the next quote on any
// pair sharing a token continues along its curve. The opposite direction is not
// credited, which only under-quotes it.
func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	i, j := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenAmountOut.Token)
	if i < 0 || j < 0 || i == j {
		return
	}
	if s.inUsed == nil {
		n := len(s.Info.Tokens)
		s.inUsed, s.outTaken = make([]uint256.Int, n), make([]uint256.Int, n)
	}
	var in, out uint256.Int
	in.SetFromBig(params.TokenAmountIn.Amount)
	out.SetFromBig(params.TokenAmountOut.Amount)
	s.inUsed[i].Add(&s.inUsed[i], &in)
	s.outTaken[j].Add(&s.outTaken[j], &out)

	s.Info.Reserves = slices.Clone(s.Info.Reserves)
	s.Info.Reserves[i] = new(big.Int).Add(s.Info.Reserves[i], params.TokenAmountIn.Amount)
	s.Info.Reserves[j] = new(big.Int).Sub(s.Info.Reserves[j], params.TokenAmountOut.Amount)
}

// GetMetaInfo returns the approval target (the core pool) and whether any
// cached leg is off the open path.
func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return PoolMeta{ApprovalAddress: s.Info.Address, BlockNumber: s.Info.BlockNumber, Gated: s.gated}
}

// GetApprovalAddress is the core pool: swap_qe pulls amountIn from msg.sender
// via transferFrom, so the executor must approve this address.
func (s *PoolSimulator) GetApprovalAddress(_, _ string) string { return s.Info.Address }

// stale reports whether the tracker snapshot is older than MaxAge.
func stale(timestamp int64) bool {
	return timestamp > 0 && time.Since(time.Unix(timestamp, 0)) > MaxAge
}
