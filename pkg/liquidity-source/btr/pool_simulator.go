package btr

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
	// used[from*n+to] is the input already swapped through the pair by
	// UpdateBalance, so a later swap in the same route continues along the
	// curve instead of restarting at the first sample. Nil until first use.
	used  []uint256.Int
	gated bool
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
		gated:   lo.SomeBy(extra.Flags, func(f uint16) bool { return gate(f) != nil }),
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

// gate mirrors the BTR open-path predicate `(flags & (HALT_MASK|SWAP_ENABLED|SWAP_GATED)) == SWAP_ENABLED`,
// but tells a lane-gated leg (SWAP_GATED) apart from a halted/disabled one.
func gate(flags uint16) error {
	if flags&HaltMask != 0 || flags&SwapEnabledBit == 0 {
		return ErrBTRGated
	}
	if flags&SwapGatedBit != 0 {
		return ErrBTRGatedLane
	}
	return nil
}

// interpolate writes the ladder output for x into res: piecewise linear between
// samples, ErrBTRAmountTooLarge above the last. Below the first sample it
// extends the first segment backwards (zero once it crosses the axis), because
// the live curve has a fixed cost that makes the dust head convex: scaling from
// the origin would overquote there by up to ~20%.
func interpolate(pts []Point, x, res *uint256.Int) error {
	if x.Gt(&pts[len(pts)-1].In) {
		return ErrBTRAmountTooLarge
	}
	if !x.Gt(&pts[0].In) {
		if len(pts) == 1 {
			big256.MulDivDown(res, &pts[0].Out, x, &pts[0].In)
			return nil
		}
		// out = p0.Out - (p0.In-x) * (p1.Out-p0.Out) / (p1.In-p0.In), rounded down
		var dy, dx, dn, cut uint256.Int
		big256.MulDivUp(&cut, dx.Sub(&pts[0].In, x), dy.Sub(&pts[1].Out, &pts[0].Out), dn.Sub(&pts[1].In, &pts[0].In))
		if cut.Gt(&pts[0].Out) {
			res.Clear()
		} else {
			res.Sub(&pts[0].Out, &cut)
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

// CalcAmountOut returns the BTR output for an exact-in amount, from the sampled
// ladder. It refuses a leg that is not on the open path, so while the live core
// is SWAP_GATED it returns no route.
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
	if err := gate(s.extra.Flags[i]); err != nil {
		return nil, err
	}
	if err := gate(s.extra.Flags[j]); err != nil {
		return nil, err
	}

	n := len(s.Info.Tokens)
	ladder := s.ladders[i*n+j]
	if ladder == nil {
		return nil, ErrBTRNoRoute
	}

	// Output for this swap = f(used+in) - f(used) along the pair's curve.
	var amountOut, x uint256.Int
	if s.used != nil && !s.used[i*n+j].IsZero() {
		var base uint256.Int
		if err := interpolate(ladder.Points, &s.used[i*n+j], &base); err != nil {
			return nil, err
		}
		if _, overflow := x.AddOverflow(&amountIn, &s.used[i*n+j]); overflow {
			return nil, ErrBTRAmountTooLarge
		}
		if err := interpolate(ladder.Points, &x, &amountOut); err != nil {
			return nil, err
		}
		amountOut.Sub(&amountOut, &base)
	} else if err := interpolate(ladder.Points, &amountIn, &amountOut); err != nil {
		return nil, err
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

// CloneState deep-copies used, the only slice UpdateBalance writes in place.
// Ladders and flags are immutable.
func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.used = slices.Clone(s.used)
	return &cloned
}

// UpdateBalance records the swapped input on the pair so the next quote on the
// pair continues along the curve. The opposite direction is not credited, which
// only under-quotes it.
func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	i, j := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenAmountOut.Token)
	if i < 0 || j < 0 || i == j {
		return
	}
	n := len(s.Info.Tokens)
	if s.used == nil {
		s.used = make([]uint256.Int, n*n)
	}
	var in uint256.Int
	in.SetFromBig(params.TokenAmountIn.Amount)
	s.used[i*n+j].Add(&s.used[i*n+j], &in)

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
