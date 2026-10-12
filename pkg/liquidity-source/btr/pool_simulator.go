package btr

import (
	"math/big"
	"time"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// PoolSimulator prices a BTR AIMM leg from the per-block sampled ladder the
// tracker built. It performs no RPC: dex-lib simulators are constructed without
// an RPC client and are called in the route-finder hot path, so an eth_call per
// CalcAmountOut is not an option. The ladder is the cached on-chain quote.
type PoolSimulator struct {
	pool.Pool
	static StaticExtra
	extra  Extra
}

var (
	_ pool.IPoolSimulator = (*PoolSimulator)(nil)
	_                     = pool.RegisterFactory(DexType, NewPoolSimulator)
)

// NewPoolSimulator is registered with pool.RegisterFactory; dex-lib passes the
// entity pool and no RPC client. The tracker is responsible for filling Extra.
func NewPoolSimulator(params pool.FactoryParams) (*PoolSimulator, error) {
	ep := params.EntityPool
	if len(ep.Tokens) < 2 {
		return nil, ErrBTRBadState
	}
	if params.Opts.StaleCheck && stale(ep.Timestamp) {
		return nil, ErrBTRBadState
	}

	var static StaticExtra
	if len(ep.StaticExtra) > 0 {
		if err := json.Unmarshal([]byte(ep.StaticExtra), &static); err != nil {
			return nil, err
		}
	}

	var extra Extra
	if len(ep.Extra) > 0 {
		if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
			return nil, err
		}
	}

	reserves := make([]*big.Int, len(ep.Tokens))
	for i, r := range ep.Reserves {
		reserve, ok := new(big.Int).SetString(r, 10)
		if !ok || reserve.Sign() < 0 {
			reserve = big.NewInt(0)
		}
		if i < len(reserves) {
			reserves[i] = reserve
		}
	}
	for i := range reserves {
		if reserves[i] == nil {
			reserves[i] = big.NewInt(0)
		}
	}

	info := pool.PoolInfo{
		Address:     lowerAddr(ep.Address),
		Exchange:    ep.Exchange,
		Type:        ep.Type,
		Tokens:      lo.Map(ep.Tokens, func(t *entity.PoolToken, _ int) string { return lowerAddr(t.Address) }),
		Reserves:    reserves,
		BlockNumber: ep.BlockNumber,
	}

	return &PoolSimulator{
		Pool:   pool.Pool{Info: info},
		static: static,
		extra:  extra,
	}, nil
}

// gate is pure so it can be unit-tested. It mirrors the BTR pool's on-chain
// gate: HALT_MASK always refuses, SWAP_ENABLED must be set, and a SWAP_GATED
// leg is admitted only when laneGranted is true (BTR governance granted this
// executor the SWAP_GATED lane 0x400). A lane does not widen the halt or the
// enable bit. It distinguishes the lane case from a halted/disabled one so the
// route finder can tell "no route until the lane is granted" from "no route".
func gate(flags uint16, laneGranted bool) error {
	if flags&HaltMask != 0 || flags&SwapEnabledBit == 0 {
		return ErrBTRGated
	}
	if flags&SwapGatedBit != 0 && !laneGranted {
		return ErrBTRGatedLane
	}
	return nil
}

// openPathAllowed is the full predicate a normal (non-lane) executor needs on
// both legs.
func openPathAllowed(flags uint16) bool { return flags&OpenPathMask == SwapEnabledBit }

// legOpen reports whether one leg is tradeable by an executor whose lane state
// is laneGranted. With the lane granted a SWAP_GATED leg is open; HALT or a
// missing SWAP_ENABLED is not.
func legOpen(flags uint16, laneGranted bool) bool { return gate(flags, laneGranted) == nil }

// legAllowed gates one leg using the cached per-token flags.
func (s *PoolSimulator) legAllowed(token string) error {
	if s.extra.Flags == nil {
		return ErrBTRBadState
	}
	flags, ok := s.extra.Flags[lowerAddr(token)]
	if !ok {
		return ErrBTRBadState
	}
	return gate(flags, s.extra.LaneGranted)
}

// CalcAmountOut returns the BTR output for an exact-in amount, from the sampled
// ladder. It refuses a leg that is not on this executor's open path, so while
// the live core is SWAP_GATED and no lane is granted it returns no route (the
// correct behaviour until BTR governance grants the lane or clears the gate).
// With LaneGranted set a SWAP_GATED leg quotes, but HALT and a missing
// SWAP_ENABLED still refuse.
func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if params.TokenAmountIn.Amount == nil || params.TokenAmountIn.Amount.Sign() <= 0 {
		return nil, ErrBTRInvalidAmount
	}
	tokenIn, tokenOut := lowerAddr(params.TokenAmountIn.Token), lowerAddr(params.TokenOut)
	if tokenIn == tokenOut {
		return nil, ErrBTRBadState
	}
	if err := s.legAllowed(tokenIn); err != nil {
		return nil, err
	}
	if err := s.legAllowed(tokenOut); err != nil {
		return nil, err
	}

	ladder := s.extra.Ladders[pairKey(tokenIn, tokenOut)]
	amountOut, err := interpolate(ladder, params.TokenAmountIn.Amount)
	if err != nil {
		return nil, err
	}
	if amountOut.Sign() <= 0 {
		return nil, ErrBTRNoFill
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: amountOut},
		Fee:            &pool.TokenAmount{Token: params.TokenAmountIn.Token, Amount: big.NewInt(0)},
		Gas:            DefaultGas,
		SwapInfo: SwapInfo{
			Pool:     s.Info.Address,
			TokenIn:  tokenIn,
			TokenOut: tokenOut,
			AmountIn: params.TokenAmountIn.Amount.String(),
		},
	}, nil
}

// CloneState returns the same simulator: the sampled ladder is immutable within
// a block and UpdateBalance is a no-op, so there is no per-swap state to clone.
func (s *PoolSimulator) CloneState() pool.IPoolSimulator { return s }

// UpdateBalance is a no-op. A BTR fill does not move the cached mark; the next
// tracker refresh re-probes the curve at the new block. A route that uses the
// BTR core on two legs (for example X->USDC then USDC->Y in a split route)
// therefore prices the second leg off an independent ladder and over-reports
// its depth. dex-lib offers no per-route signal a simulator can reject on, so
// this is documented, not enforced: the KyberSwap router's min-return check
// reverts the whole route when the over-quote is not delivered on chain.
func (s *PoolSimulator) UpdateBalance(_ pool.UpdateBalanceParams) {}

// GetMetaInfo returns the approval target (the core pool) and whether any
// cached leg is not on this executor's open path.
func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	gated := false
	for _, flags := range s.extra.Flags {
		if !legOpen(flags, s.extra.LaneGranted) {
			gated = true
			break
		}
	}
	return PoolMeta{
		ApprovalAddress: s.Info.Address,
		BlockNumber:     s.Info.BlockNumber,
		Gated:           gated,
	}
}

// GetApprovalAddress is the core pool: swap_qe pulls amountIn from msg.sender
// via transferFrom, so the executor must approve this address.
func (s *PoolSimulator) GetApprovalAddress(_, _ string) string { return s.Info.Address }

// Stale reports whether the cached ladder is older than the accepted age. The
// tracker stamps Timestamp, which the simulator does not keep; callers that
// need staleness use the entity pool timestamp upstream. Kept for parity with
// ladder-quoted pools.
func stale(timestamp int64) bool {
	return timestamp > 0 && time.Since(time.Unix(timestamp, 0)) > MaxAge
}
