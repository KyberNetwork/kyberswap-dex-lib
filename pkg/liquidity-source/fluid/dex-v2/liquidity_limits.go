package dexv2

import (
	"math/big"
	"strings"

	"github.com/KyberNetwork/logger"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	liquidityFloat = iota
	liquidityWithdraw
	liquidityBorrow
	liquidityDebt
	liquidityCash
	liquiditySupplyAllowed
	liquidityWithdrawAllowed
	liquidityBorrowAllowed
	liquidityPaybackAllowed
	liquidityInputTotal
	liquidityOutputTotal
	liquidityOutputReserved
	liquidityRoundingBuffer
	liquidityGrowth
	liquidityDepositSafe
	liquidityDepositHeadroom
	liquidityFieldCount
)

var liquidityFieldNames = [liquidityFieldCount]string{"float", "withdraw", "borrow", "debt", "cash", "supplyAllowed", "withdrawAllowed", "borrowAllowed", "paybackAllowed", "inputTotal", "outputTotal", "outputReserved", "roundingBuffer", "growth", "depositSafe", "depositHeadroom"}

// LiquidityLimits holds conservative shared-vault budgets. Endpoint totals are
// cumulative because the executor can combine several paths into one settle.
type LiquidityLimits struct {
	Float           uint256.Int `json:"float"`
	Withdraw        uint256.Int `json:"withdraw"`
	Borrow          uint256.Int `json:"borrow"`
	Debt            uint256.Int `json:"debt"`
	Cash            uint256.Int `json:"cash"`
	Allowed         [4]bool     `json:"allowed"`
	InputTotal      uint256.Int `json:"inputTotal"`
	OutputTotal     uint256.Int `json:"outputTotal"`
	OutputReserved  uint256.Int `json:"outputReserved"`
	RoundingBuffer  uint256.Int `json:"roundingBuffer"`
	Growth          uint256.Int `json:"growth"`
	DepositSafe     bool        `json:"depositSafe"`
	DepositHeadroom uint256.Int `json:"depositHeadroom"`
}

type liquidityUsage struct {
	Withdraw, Borrow, Debt, Cash, Growth uint256.Int
	Input, Output, OutputReserved        uint256.Int
	DepositHeadroom                      uint256.Int
}

func (s *LiquidityLimits) values() (v [liquidityFieldCount]uint256.Int) {
	v[0], v[1], v[2], v[3], v[4] = s.Float, s.Withdraw, s.Borrow, s.Debt, s.Cash
	for i, allowed := range s.Allowed {
		if allowed {
			v[5+i].SetOne()
		}
	}
	v[9], v[10], v[11], v[12], v[13] = s.InputTotal, s.OutputTotal, s.OutputReserved, s.RoundingBuffer, s.Growth
	v[liquidityDepositHeadroom] = s.DepositHeadroom
	if s.DepositSafe {
		v[liquidityDepositSafe].SetOne()
	}
	return
}

var _ pool.ISwapLimitMergePolicy = (*PoolSimulator)(nil)

func liquidityMergePolicy(field int) pool.SwapLimitMergePolicy {
	switch field {
	case liquidityInputTotal, liquidityOutputTotal, liquidityOutputReserved, liquidityRoundingBuffer:
		return pool.SwapLimitMergeMax
	default:
		return pool.SwapLimitMergeMin
	}
}

func (p *PoolSimulator) GetLimitMergePolicy(key string) pool.SwapLimitMergePolicy {
	kind := key[strings.LastIndexByte(key, ':')+1:]
	for field, name := range liquidityFieldNames {
		if kind == name {
			return liquidityMergePolicy(field)
		}
	}
	// Unknown keys must never enlarge shared spending capacity.
	return pool.SwapLimitMergeMin
}

func (p *PoolSimulator) liquidityKey(i, field int) string {
	return p.liquidityKeys[i] + liquidityFieldNames[field]
}

func (p *PoolSimulator) liquidityValues(i int, limit pool.SwapLimit) ([liquidityFieldCount]uint256.Int, error) {
	v := p.liquidityLimits[i].values()
	if limit != nil {
		for field := range v {
			shared := limit.GetLimit(p.liquidityKey(i, field))
			var x uint256.Int
			if shared == nil || shared.Sign() < 0 || x.SetFromBig(shared) {
				return v, ErrUnsupportedLiquidityState
			}
			policy := liquidityMergePolicy(field)
			if (policy == pool.SwapLimitMergeMax && x.Gt(&v[field])) || (policy == pool.SwapLimitMergeMin && x.Lt(&v[field])) {
				v[field] = x
			}
		}
	}
	return v, nil
}

func reserveLiquidity(budget, spent, amount *uint256.Int) error {
	if amount.Gt(budget) {
		return ErrUnsupportedLiquidityState
	}
	budget.Sub(budget, amount)
	if _, overflow := spent.AddOverflow(spent, amount); overflow {
		return ErrOverflow
	}
	return nil
}

func (s *LiquidityLimits) padded(amount *uint256.Int) (result uint256.Int, err error) {
	if _, overflow := result.AddOverflow(amount, &s.RoundingBuffer); overflow {
		err = ErrOverflow
	}
	return
}

func validateEndpoints(v *[liquidityFieldCount]uint256.Int) error {
	var remainingFloat uint256.Int
	if v[liquidityFloat].Gt(&v[liquidityOutputTotal]) {
		remainingFloat.Sub(&v[liquidityFloat], &v[liquidityOutputTotal])
	}
	if v[liquidityInputTotal].Gt(&remainingFloat) && v[liquiditySupplyAllowed].IsZero() {
		return ErrUnsupportedLiquidityState
	}
	if v[liquidityOutputTotal].Gt(&v[liquidityFloat]) {
		if v[liquidityWithdrawAllowed].IsZero() || v[liquidityOutputReserved].Gt(&v[liquidityWithdraw]) || v[liquidityOutputReserved].Gt(&v[liquidityCash]) {
			return ErrUnsupportedLiquidityState
		}
	}
	return nil
}

// An initial nonzero floor L0 satisfies L0 >= (1-expandPercent)*S0.
// Every later floor is bounded by L0 plus cumulative deposits, even if a
// withdrawal resets it to zero. No existing or newly generated decay credit
// is assumed. Initial zero floors instead require a sub-base deposit budget.
func reserveDeposit(v *[liquidityFieldCount]uint256.Int, usage *liquidityUsage, amount *uint256.Int) error {
	if v[liquidityDepositSafe].IsZero() {
		if amount.Cmp(&v[liquidityDepositHeadroom]) <= 0 {
			return reserveLiquidity(&v[liquidityDepositHeadroom], &usage.DepositHeadroom, amount)
		}
		// Once a deposit can cross the base threshold, no old withdrawal
		// headroom is guaranteed; discard the remaining sub-base budget too.
		remaining := v[liquidityDepositHeadroom]
		if err := reserveLiquidity(&v[liquidityDepositHeadroom], &usage.DepositHeadroom, &remaining); err != nil {
			return err
		}
	}
	debit := *amount
	if v[liquidityDepositSafe].IsZero() || debit.Gt(&v[liquidityWithdraw]) {
		debit = v[liquidityWithdraw]
	}
	return reserveLiquidity(&v[liquidityWithdraw], &usage.Withdraw, &debit)
}

func addLiquidityTotal(v *uint256.Int, amount *uint256.Int) error {
	if _, overflow := v.AddOverflow(v, amount); overflow {
		return ErrOverflow
	}
	return nil
}

func (p *PoolSimulator) quoteLiquidity(limit pool.SwapLimit, in, out int, dexType uint32, amountIn, amountOut, fee *uint256.Int) (usage [2]liquidityUsage, err error) {
	if !p.staticExtra.TrackLiquidityLimits {
		return
	}
	if dexType != D3_MODULE && !p.extra.SupplyOnlySettlement {
		return usage, ErrUnsupportedLiquidityState
	}
	input, err := p.liquidityValues(in, limit)
	if err != nil {
		return usage, err
	}
	output, err := p.liquidityValues(out, limit)
	if err != nil {
		return usage, err
	}
	inputBudget := p.liquidityLimits[in]
	inputBudget.RoundingBuffer = input[liquidityRoundingBuffer]
	outputBudget := p.liquidityLimits[out]
	outputBudget.RoundingBuffer = output[liquidityRoundingBuffer]
	// Every operate leg remains below the threshold at which Liquidity applies
	// its aggregate ratio check. Requiring more than a storage quantum ensures
	// the stored user position actually changes after rounding.
	var maxLeg uint256.Int
	maxLeg.SetOne().Lsh(&maxLeg, 79)
	var grossOut uint256.Int
	if _, overflow := grossOut.AddOverflow(amountOut, fee); overflow {
		return usage, ErrOverflow
	}
	if amountIn.Cmp(&inputBudget.RoundingBuffer) <= 0 || amountOut.Cmp(&outputBudget.RoundingBuffer) <= 0 ||
		amountIn.Gt(&maxLeg) || grossOut.Gt(&maxLeg) {
		return usage, ErrUnsupportedLiquidityState
	}

	inputPadded, err := inputBudget.padded(amountIn)
	if err != nil {
		return usage, err
	}
	outputPadded, err := outputBudget.padded(amountOut)
	if err != nil {
		return usage, err
	}
	grossPadded, err := outputBudget.padded(&grossOut)
	if err != nil {
		return usage, err
	}
	if err = reserveLiquidity(&input[liquidityGrowth], &usage[in].Growth, &inputPadded); err != nil {
		return usage, err
	}
	if err = reserveLiquidity(&output[liquidityGrowth], &usage[out].Growth, &grossPadded); err != nil {
		return usage, err
	}
	if dexType != D3_MODULE {
		var gross uint256.Int
		if _, overflow := gross.AddOverflow(amountOut, fee); overflow {
			return usage, ErrOverflow
		}
		gross, err = outputBudget.padded(&gross)
		if err != nil {
			return usage, err
		}
		if input[liquidityPaybackAllowed].IsZero() || output[liquidityBorrowAllowed].IsZero() {
			return usage, ErrUnsupportedLiquidityState
		}
		if err = reserveLiquidity(&input[liquidityDebt], &usage[in].Debt, &inputPadded); err != nil {
			return usage, err
		}
		if err = reserveLiquidity(&output[liquidityBorrow], &usage[out].Borrow, &gross); err != nil {
			return usage, err
		}
		if p.extra.SupplyOnlySettlement {
			// SC PRs #781/#987 convert D4 pending borrow to supply after each hop:
			// tokenIn: S=B=-amountIn; tokenOut: S=B=amountOut+protocolFee+lpFee.
			// These equal-leg settles call Liquidity even if DEX float can fund
			// the final supply-only endpoint transfer.
			if input[liquidityWithdrawAllowed].IsZero() || output[liquiditySupplyAllowed].IsZero() {
				return usage, ErrUnsupportedLiquidityState
			}
			if err = reserveLiquidity(&input[liquidityWithdraw], &usage[in].Withdraw, &inputPadded); err != nil {
				return usage, err
			}
			// Withdrawal can reduce utilization headroom by its full amount.
			// Payback can shrink the expanded borrow ceiling by up to twice
			// the payback (expand percent is at most 100%). Do not credit
			// the repaid debt, and reserve all three amounts.
			var debit uint256.Int
			factor := uint256.NewInt(3)
			if _, overflow := debit.MulOverflow(&inputPadded, factor); overflow {
				return usage, ErrOverflow
			}
			if debit.Gt(&input[liquidityBorrow]) {
				debit = input[liquidityBorrow]
			}
			if err = reserveLiquidity(&input[liquidityBorrow], &usage[in].Borrow, &debit); err != nil {
				return usage, err
			}
		}
	}
	if dexType == D3_MODULE || p.extra.SupplyOnlySettlement {
		// Conservatively account for endpoint deposits even when float may absorb
		// them. D4 also deposits gross output during S=B conversion.
		if err = reserveDeposit(&input, &usage[in], &inputPadded); err != nil {
			return usage, err
		}
		if dexType != D3_MODULE {
			paddedGross, e := outputBudget.padded(&grossOut)
			if e != nil {
				return usage, e
			}
			if err = reserveDeposit(&output, &usage[out], &paddedGross); err != nil {
				return usage, err
			}
		}
		usage[in].Input = *amountIn
		usage[out].Output = *amountOut
		usage[out].OutputReserved = outputPadded
		if err = addLiquidityTotal(&input[liquidityInputTotal], amountIn); err != nil {
			return usage, err
		}
		if err = addLiquidityTotal(&output[liquidityOutputTotal], amountOut); err != nil {
			return usage, err
		}
		if err = addLiquidityTotal(&output[liquidityOutputReserved], &outputPadded); err != nil {
			return usage, err
		}
	}
	if input[liquidityInputTotal].Gt(&maxLeg) || output[liquidityOutputTotal].Gt(&maxLeg) {
		return usage, ErrUnsupportedLiquidityState
	}
	// Supply withdrawals can reduce max-utilization borrow headroom. Debit the
	// full possible withdrawal (plus raw-share rounding), even if float may pay it.
	debit := outputPadded
	if debit.Gt(&output[liquidityBorrow]) {
		debit = output[liquidityBorrow]
	}
	if err = reserveLiquidity(&output[liquidityBorrow], &usage[out].Borrow, &debit); err != nil {
		return usage, err
	}
	if err = validateEndpoints(&input); err != nil {
		return usage, err
	}
	err = validateEndpoints(&output)
	return
}

func (p *PoolSimulator) CalculateLimit() map[string]*big.Int {
	if !p.staticExtra.TrackLiquidityLimits {
		return nil
	}
	result := make(map[string]*big.Int, 2*liquidityFieldCount)
	for i := range 2 {
		for field, value := range p.liquidityLimits[i].values() {
			result[p.liquidityKey(i, field)] = value.ToBig()
		}
	}
	return result
}

func updateLiquidityLimit(limit pool.SwapLimit, key string, amount *uint256.Int, increase bool) {
	if limit == nil || amount.IsZero() {
		return
	}
	var zero big.Int
	var err error
	if increase {
		_, _, err = limit.UpdateLimit(key, key, &zero, amount.ToBig())
	} else {
		_, _, err = limit.UpdateLimit(key, key, amount.ToBig(), &zero)
	}
	if err != nil {
		logger.Warnf("failed to update Fluid liquidity limit: %v", err)
	}
}

func (p *PoolSimulator) applyLiquidity(usage [2]liquidityUsage, limit pool.SwapLimit) {
	if !p.staticExtra.TrackLiquidityLimits {
		return
	}
	for i := range 2 {
		s, u := &p.liquidityLimits[i], &usage[i]
		for _, field := range []struct {
			kind        int
			dst, amount *uint256.Int
		}{{liquidityWithdraw, &s.Withdraw, &u.Withdraw}, {liquidityBorrow, &s.Borrow, &u.Borrow}, {liquidityDebt, &s.Debt, &u.Debt}, {liquidityCash, &s.Cash, &u.Cash}, {liquidityGrowth, &s.Growth, &u.Growth}, {liquidityDepositHeadroom, &s.DepositHeadroom, &u.DepositHeadroom}} {
			field.dst.Sub(field.dst, field.amount)
			updateLiquidityLimit(limit, p.liquidityKey(i, field.kind), field.amount, false)
		}
		for _, field := range []struct {
			kind        int
			dst, amount *uint256.Int
		}{{liquidityInputTotal, &s.InputTotal, &u.Input}, {liquidityOutputTotal, &s.OutputTotal, &u.Output}, {liquidityOutputReserved, &s.OutputReserved, &u.OutputReserved}} {
			field.dst.Add(field.dst, field.amount)
			updateLiquidityLimit(limit, p.liquidityKey(i, field.kind), field.amount, true)
		}
	}
}
