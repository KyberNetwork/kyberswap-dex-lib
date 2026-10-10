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

// gate is pure so it can be unit-tested. It mirrors the BTR pool's open-path
// predicate `(flags & OpenPathMask) == SwapEnabledBit`, but distinguishes a
// lane-gated leg (SWAP_GATED) from a halted/disabled one for the route finder.
func gate(flags uint16) error {
	if flags&HaltMask != 0 || flags&SwapEnabledBit == 0 {
		return ErrBTRGated
	}
	if flags&SwapGatedBit != 0 {
		return ErrBTRGatedLane
	}
	return nil
}

// openPathAllowed is the full predicate a normal (non-lane) executor needs on
// both legs.
func openPathAllowed(flags uint16) bool { return flags&OpenPathMask == SwapEnabledBit }

// legAllowed gates one leg using the cached per-token flags.
func (s *PoolSimulator) legAllowed(token string) error {
	if s.extra.Flags == nil {
		return ErrBTRBadState
	}
	flags, ok := s.extra.Flags[lowerAddr(token)]
	if !ok {
		return ErrBTRBadState
	}
	return gate(flags)
}

// CalcAmountOut returns the BTR output for an exact-in amount, from the sampled
// ladder. It refuses a leg that is not on the open path, so while the live core
// is SWAP_GATED it returns no route (the correct behaviour until the gate is lifted).
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
// tracker refresh re-probes the curve at the new block. Route finders that
// apply several swaps to one pool within a route will over-report depth for the
// later leg, which the M1 simulate-at-execution check catches.
func (s *PoolSimulator) UpdateBalance(_ pool.UpdateBalanceParams) {}

// GetMetaInfo returns the approval target (the core pool) and whether every
// cached leg is on the open path.
func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	gated := false
	for _, flags := range s.extra.Flags {
		if !openPathAllowed(flags) {
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
