package gohm

import (
	"math/big"
	"strings"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// PoolSimulator implements pool.IPoolSimulator for the gOHM (OlympusDAO staking) protocol.
// It supports six swap directions between OHM, sOHM, and gOHM using index-based pricing.
//
// Pricing:
//   - OHM <-> sOHM: 1:1 (both 9-decimal rebasing tokens)
//   - OHM/sOHM -> gOHM: balanceTo(amount, index) = amount * 1e18 / index
//   - gOHM -> OHM/sOHM: balanceFrom(amount, index) = amount * index / 1e18
//
// Binding constraints (enforced per CalcAmountOut):
//   - OHM->sOHM:  amountIn  <= SOHMReserve (staking must hold enough sOHM to transfer out 1:1)
//   - sOHM->OHM:  amountIn  <= OHMReserve  (on-chain require(amount_ <= OHM.balanceOf(this)))
//   - gOHM->OHM:  balanceFrom(amountIn) <= OHMReserve
//   - gOHM->sOHM: balanceFrom(amountIn) <= SOHMReserve
//   - OHM->gOHM:  no staking-balance cap (gOHM.mint is uncapped)
//   - sOHM->gOHM: no staking-balance cap (sOHM circulating supply, not staking balance)
//
// Token positions in pool.Info.Tokens — set by the lister, never reordered.
const (
	idxOHM  = 0
	idxSOHM = 1
	idxGOHM = 2
)

type PoolSimulator struct {
	pool.Pool

	index        *uint256.Int
	warmupPeriod uint64

	// ohmReserve is OHM.balanceOf(staking); sohmReserve is sOHM.balanceOf(staking).
	// Both are fetched by the tracker each refresh cycle and used to enforce
	// binding liquidity caps in CalcAmountOut.
	ohmReserve  *uint256.Int
	sohmReserve *uint256.Int

	// scratch and scratchIn are pre-allocated uint256.Ints reused across
	// CalcAmountOut calls to avoid per-call heap allocations. scratchIn holds the
	// parsed amountIn; scratch holds the intermediate/final result. Neither must
	// escape across goroutines or be shared between a simulator and its clone.
	scratch   uint256.Int
	scratchIn uint256.Int
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	var extra PoolExtra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}

	// Resolve reserve fields. Pools created before the balance-cap change will
	// have nil OHMReserve/SOHMReserve in their serialized Extra. Use zero as a
	// safe sentinel — the cap check treats zero-reserve as "no liquidity available"
	// which correctly blocks routing until the tracker refreshes with real values.
	ohmReserve := extra.OHMReserve
	if ohmReserve == nil {
		ohmReserve = new(uint256.Int)
	}
	sohmReserve := extra.SOHMReserve
	if sohmReserve == nil {
		sohmReserve = new(uint256.Int)
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     strings.ToLower(ep.Address),
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      lo.Map(ep.Tokens, func(item *entity.PoolToken, _ int) string { return item.Address }),
			Reserves:    lo.Map(ep.Reserves, func(item string, _ int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: ep.BlockNumber,
		}},
		index:        extra.Index,
		warmupPeriod: extra.WarmupPeriod,
		ohmReserve:   ohmReserve,
		sohmReserve:  sohmReserve,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn := strings.ToLower(params.TokenAmountIn.Token)
	tokenOut := strings.ToLower(params.TokenOut)
	// Use scratchIn to parse amountIn without a heap allocation.
	// SetFromBig returns true on overflow (amount > 2^256-1), which cannot occur for
	// realistic token amounts; treat it the same as invalid input.
	if overflow := s.scratchIn.SetFromBig(params.TokenAmountIn.Amount); overflow {
		return nil, ErrInvalidTokenIn
	}
	amountIn := &s.scratchIn

	if amountIn.IsZero() {
		return nil, ErrZeroAmount
	}

	action, err := ActionFor(tokenIn, tokenOut, s.Info.Tokens[idxOHM], s.Info.Tokens[idxSOHM], s.Info.Tokens[idxGOHM])
	if err != nil {
		return nil, err
	}

	if action == ActionStakeToSOHM || action == ActionUnstakeSOHM {
		// 1:1 swap — no index arithmetic needed.
		// Cap check: staking must hold enough of the output token.
		//   OHM->sOHM: require amountIn <= sOHM.balanceOf(staking)
		//   sOHM->OHM: require amountIn <= OHM.balanceOf(staking)
		if action == ActionStakeToSOHM {
			if amountIn.Gt(s.sohmReserve) {
				return nil, ErrInsufficientLiquidity
			}
		} else {
			if amountIn.Gt(s.ohmReserve) {
				return nil, ErrInsufficientLiquidity
			}
		}
		gasEstimate := dfGas.Stake
		if action == ActionUnstakeSOHM {
			gasEstimate = dfGas.Unstake
		}
		return &pool.CalcAmountOutResult{
			TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: params.TokenAmountIn.Amount},
			Fee:            &pool.TokenAmount{Token: params.TokenOut, Amount: bignumber.ZeroBI},
			Gas:            gasEstimate,
			SwapInfo:       SwapInfo{Action: action},
		}, nil
	}

	if s.warmupPeriod > 0 {
		return nil, ErrWarmupActive
	}

	var gasEstimate int64

	if s.index.IsZero() {
		return nil, ErrIndexZero
	}

	switch action {
	case ActionStakeToGOHM:
		// OHM->gOHM (stake rebasing=false): no staking-balance cap.
		// gOHM.mint is uncapped — the only limit is OHM total supply.
		// balanceTo: result = amountIn * 1e18 / index
		s.scratch.Mul(amountIn, number_1e18)
		s.scratch.Div(&s.scratch, s.index)
		gasEstimate = dfGas.Stake
	case ActionWrap:
		// sOHM->gOHM (wrap): no staking-balance cap.
		// Limited by sOHM circulating supply, not staking balance.
		// balanceTo: result = amountIn * 1e18 / index
		s.scratch.Mul(amountIn, number_1e18)
		s.scratch.Div(&s.scratch, s.index)
		gasEstimate = dfGas.Wrap
	case ActionUnstakeGOHM:
		// gOHM->OHM (unstake rebasing=false): cap on outbound OHM.
		// On-chain: require(amount_ <= OHM.balanceOf(this))
		// The OHM output = balanceFrom(amountIn) = amountIn * index / 1e18.
		// Compute output first, then cap-check.
		s.scratch.Mul(amountIn, s.index)
		s.scratch.Div(&s.scratch, number_1e18)
		if s.scratch.Gt(s.ohmReserve) {
			return nil, ErrInsufficientLiquidity
		}
		gasEstimate = dfGas.Unstake
	case ActionUnwrap:
		// gOHM->sOHM (unwrap): cap on outbound sOHM.
		// Staking safeTransfers sOHM to recipient; must hold enough.
		// The sOHM output = balanceFrom(amountIn) = amountIn * index / 1e18.
		// Compute output first, then cap-check.
		s.scratch.Mul(amountIn, s.index)
		s.scratch.Div(&s.scratch, number_1e18)
		if s.scratch.Gt(s.sohmReserve) {
			return nil, ErrInsufficientLiquidity
		}
		gasEstimate = dfGas.Unwrap
	default:
		return nil, ErrInvalidTokenIn
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: s.scratch.ToBig()},
		Fee:            &pool.TokenAmount{Token: params.TokenOut, Amount: bignumber.ZeroBI},
		Gas:            gasEstimate,
		SwapInfo:       SwapInfo{Action: action},
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	info, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		return
	}
	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow {
		return
	}
	amountOut, overflow := uint256.FromBig(params.TokenAmountOut.Amount)
	if overflow {
		return
	}

	switch info.Action {
	case ActionStakeToSOHM: // OHM in, sOHM out (1:1).
		s.ohmReserve.Add(s.ohmReserve, amountIn)
		s.sohmReserve.Sub(s.sohmReserve, amountOut)
	case ActionUnstakeSOHM: // sOHM in, OHM out (1:1).
		s.sohmReserve.Add(s.sohmReserve, amountIn)
		s.ohmReserve.Sub(s.ohmReserve, amountOut)
	case ActionStakeToGOHM: // OHM in (held by staking), gOHM out (minted externally).
		s.ohmReserve.Add(s.ohmReserve, amountIn)
	case ActionUnstakeGOHM: // gOHM in (burned externally), OHM out.
		s.ohmReserve.Sub(s.ohmReserve, amountOut)
	case ActionWrap: // sOHM in (held by staking), gOHM out (minted externally).
		s.sohmReserve.Add(s.sohmReserve, amountIn)
	case ActionUnwrap: // gOHM in (burned externally), sOHM out.
		s.sohmReserve.Sub(s.sohmReserve, amountOut)
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	if s.index != nil {
		cloned.index = new(uint256.Int).Set(s.index)
	}
	if s.ohmReserve != nil {
		cloned.ohmReserve = new(uint256.Int).Set(s.ohmReserve)
	}
	if s.sohmReserve != nil {
		cloned.sohmReserve = new(uint256.Int).Set(s.sohmReserve)
	}
	if len(s.Info.Reserves) > 0 {
		cloned.Info.Reserves = make([]*big.Int, len(s.Info.Reserves))
		for i, r := range s.Info.Reserves {
			cloned.Info.Reserves[i] = new(big.Int).Set(r)
		}
	}
	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) any {
	return PoolMeta{
		BlockNumber: s.Info.BlockNumber,
		OHM:         s.Info.Tokens[idxOHM],
		SOHM:        s.Info.Tokens[idxSOHM],
		GOHM:        s.Info.Tokens[idxGOHM],
	}
}

func ActionFor(tokenIn, tokenOut, ohm, sohm, gohm string) (Action, error) {
	switch tokenIn {
	case ohm:
		switch tokenOut {
		case sohm:
			return ActionStakeToSOHM, nil
		case gohm:
			return ActionStakeToGOHM, nil
		}
		return 0, ErrInvalidTokenOut
	case sohm:
		switch tokenOut {
		case ohm:
			return ActionUnstakeSOHM, nil
		case gohm:
			return ActionWrap, nil
		}
		return 0, ErrInvalidTokenOut
	case gohm:
		switch tokenOut {
		case ohm:
			return ActionUnstakeGOHM, nil
		case sohm:
			return ActionUnwrap, nil
		}
		return 0, ErrInvalidTokenOut
	}
	return 0, ErrInvalidTokenIn
}
