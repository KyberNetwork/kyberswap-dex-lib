package rangepool

import (
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/base"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/hooks"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var _ = pool.RegisterFactory(DexType, NewPoolSimulator)

// NewPoolSimulator wires a Range Pool onto balancer/v3's base.PoolSimulator + vault flow
// (Locked decision 2: reuse the validated scale->fee->curve->unscale vault machinery).
// The only Range-specific piece is the swapper below: OnSwap runs the weighted-product
// curve on VIRTUAL balances and caps by the fact balance.
//
// The RangePoolHook sets no dynamic-swap-fee flag and no enableHookAdjustedAmounts, so it
// cannot alter a quote (reference §4) — we pass an explicit no-op hook.
func NewPoolSimulator(params pool.FactoryParams) (*base.PoolSimulator, error) {
	entityPool := params.EntityPool

	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	return base.NewPoolSimulator(params, extra.Extra, &staticExtra.StaticExtra, &PoolSimulator{
		virtualBalances:    extra.VirtualBalances,
		normalizedWeights:  staticExtra.NormalizedWeights,
		minimumTradeAmount: extra.MinimumTradeAmount,
	}, hooks.NewNoOpHook())
}

// PoolSimulator is the Range-specific swapper embedded in base.PoolSimulator (the same
// role weighted.PoolSimulator plays for the weighted pool type). It holds the two vectors
// the Range curve prices on that the generic vault does NOT carry: the virtual balances
// and the normalized weights.
type PoolSimulator struct {
	// virtualBalances are the concentrated-liquidity curve's virtual reserves
	// (scaled18, token registration order). The vault passes FACT balances in
	// PoolSwapParams.BalancesScaled18; Range prices on these instead and only uses
	// the fact balance to cap/guard the output.
	virtualBalances []*uint256.Int

	// normalizedWeights are the immutable pool weights (scaled18, sum = 1e18).
	normalizedWeights []*uint256.Int

	// minimumTradeAmount is the Vault's minimum trade amount (scaled18). Range's custom
	// Vault uses a non-canonical value (1e12 on mainnet) that the shared vault flow —
	// hardcoded to Balancer's 1e6 — does not enforce, so OnSwap applies it directly at
	// the two _ensureValidSwapAmount points of Vault._swap. May be nil for legacy
	// snapshots; then only the shared 1e6 floor applies.
	minimumTradeAmount *uint256.Int
}

func (p *PoolSimulator) BaseGas() int64 {
	return baseGas
}

// OnSwap is the per-curve step the vault invokes after it has scaled the raw amount to
// scaled18 and (for EXACT_IN) deducted the static swap fee. It is a faithful port of
// RangePool.onSwap:
//
//   - EXACT_IN: amountOut = calcOutGivenIn(vIn, wIn, vOut, wOut, amountInAfterFee, factOut),
//     which internally caps at factOut - ABSOLUTE_MIN_TOKEN_BALANCE.
//   - EXACT_OUT: guard (amountOut + 1e6 > factOut) and (amountOut >= vOut) — either reverts
//     on-chain, so we return no price — then amountIn = calcInGivenOut(...).
//
// factOut is the fact (live) out balance, which the vault supplies as
// param.BalancesScaled18[IndexOut]. We deliberately do NOT mutate the virtual balances
// here: CalcAmountOut/CalcAmountIn must be pure, and Strategy A re-reads authoritative
// virtual balances every tracker cycle.
func (p *PoolSimulator) OnSwap(param shared.PoolSwapParams) (*uint256.Int, error) {
	if param.IndexIn >= len(p.virtualBalances) || param.IndexOut >= len(p.virtualBalances) {
		return nil, ErrInvalidToken
	}

	virtualBalanceIn := p.virtualBalances[param.IndexIn]
	virtualBalanceOut := p.virtualBalances[param.IndexOut]
	weightIn := p.normalizedWeights[param.IndexIn]
	weightOut := p.normalizedWeights[param.IndexOut]
	factBalanceOut := param.BalancesScaled18[param.IndexOut]

	// Vault._swap _ensureValidSwapAmount #1: the (post-fee for EXACT_IN) given amount.
	// The vault hands us param.AmountGivenScaled18 already fee-deducted for EXACT_IN and
	// as the raw output for EXACT_OUT — exactly what the on-chain check at line 391 sees.
	if p.belowMinimumTrade(param.AmountGivenScaled18) {
		return nil, ErrTradeAmountTooSmall
	}

	var amountCalculated *uint256.Int
	var err error

	if param.Kind == shared.ExactIn {
		amountCalculated, err = CalcOutGivenIn(
			virtualBalanceIn, weightIn,
			virtualBalanceOut, weightOut,
			param.AmountGivenScaled18, factBalanceOut,
		)
	} else {
		// EXACT_OUT: param.AmountGivenScaled18 is the requested (scaled18) output.
		amountOut := param.AmountGivenScaled18
		if !IsExactOutAllowed(amountOut, factBalanceOut, virtualBalanceOut) {
			return nil, ErrExactOutNotAllowed
		}
		amountCalculated, err = CalcInGivenOut(
			virtualBalanceIn, weightIn,
			virtualBalanceOut, weightOut,
			amountOut,
		)
	}
	if err != nil {
		return nil, err
	}

	// Vault._swap _ensureValidSwapAmount #2: the calculated amount (line 396).
	if p.belowMinimumTrade(amountCalculated) {
		return nil, ErrTradeAmountTooSmall
	}

	return amountCalculated, nil
}

// belowMinimumTrade reports whether a scaled18 amount is under the Vault's minimum trade
// amount. Nil means unknown (legacy snapshot) — the shared vault's 1e6 floor still applies,
// so we don't add a stricter gate here.
func (p *PoolSimulator) belowMinimumTrade(amountScaled18 *uint256.Int) bool {
	return p.minimumTradeAmount != nil && amountScaled18.Lt(p.minimumTradeAmount)
}
