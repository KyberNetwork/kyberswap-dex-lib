package pool

import (
	"math/big"

	"github.com/holiman/uint256"
)

// PoolSimFactoryFn builds a pool that implements IPoolSim.
type PoolSimFactoryFn func(FactoryParams) (IPoolSim, error)

// FactoryPoolSim returns the factory for a pool type as an IPoolSim. A registered pool that already implements IPoolSim
// is returned as-is. A legacy pool is wrapped by legacyAdapter.
func FactoryPoolSim(poolType string) PoolSimFactoryFn {
	factory := Factory(poolType)
	if factory == nil {
		return nil
	}
	return func(params FactoryParams) (IPoolSim, error) {
		sim, err := factory(params)
		if err != nil {
			return nil, err
		}
		if native, ok := sim.(ICalcAmtOut); ok {
			if u, ok := native.(IPoolSim); ok {
				return u, nil
			}
		}
		return legacyAdapter{IPoolSimulator: sim}, nil
	}
}

// legacyAdapter exposes a legacy big.Int IPoolSimulator as IPoolSim. Each uint256 call converts to big.Int, calls
// the legacy method, and converts the result back.
type legacyAdapter struct {
	IPoolSimulator
}

func (a legacyAdapter) CalcAmtOut(params CalcAmtOutParams) (*CalcAmtOutResult, error) {
	res, err := a.IPoolSimulator.CalcAmountOut(CalcAmountOutParams{
		TokenAmountIn: tokenAmountFromAmt(&params.TokenAmountIn),
		TokenOut:      params.TokenOut,
		Limit:         params.Limit,
	})
	if err != nil || res == nil {
		return nil, err
	}
	return calcAmtOutResultFromBig(res), nil
}

func (a legacyAdapter) CloneState() IPoolSim {
	cloned := a.IPoolSimulator.CloneState()
	if cloned == nil {
		return nil
	}
	return legacyAdapter{IPoolSimulator: cloned}
}

func (a legacyAdapter) UpdateState(params UpdateStateParams) {
	a.IPoolSimulator.UpdateBalance(UpdateBalanceParams{
		TokenAmountIn:  tokenAmountFromAmt(&params.TokenAmountIn),
		TokenAmountOut: tokenAmountFromAmt(&params.TokenAmountOut),
		Fee:            tokenAmountFromAmt(&params.Fee),
		SwapInfo:       params.SwapInfo,
		SwapLimitBig:   params.SwapLimitBig,
	})
}

func (a legacyAdapter) CalcLimit() map[string]*uint256.Int {
	return limitsToU256(a.IPoolSimulator.CalculateLimit())
}

func (a legacyAdapter) GetReserves() []uint256.Int {
	reserves := a.IPoolSimulator.GetReserves()
	out := make([]uint256.Int, len(reserves))
	for i, r := range reserves {
		out[i] = *bigToU256OrZero(r)
	}
	return out
}

func tokenAmountFromAmt(t *TokenAmt) TokenAmount {
	return TokenAmount{Token: t.Token, Amount: t.Amount.ToBig(), AmountUsd: t.AmountUsd}
}

func tokenAmountPtrToAmt(t *TokenAmount) *TokenAmt {
	if t == nil {
		return nil
	}
	out := &TokenAmt{Token: t.Token, AmountUsd: t.AmountUsd}
	out.Amount = *bigToU256OrZero(t.Amount)
	return out
}

func calcAmtOutResultFromBig(r *CalcAmountOutResult) *CalcAmtOutResult {
	return &CalcAmtOutResult{
		TokenAmountOut:         tokenAmountPtrToAmt(r.TokenAmountOut),
		Fee:                    tokenAmountPtrToAmt(r.Fee),
		RemainingTokenAmountIn: tokenAmountPtrToAmt(r.RemainingTokenAmountIn),
		Gas:                    r.Gas,
		SwapInfo:               r.SwapInfo,
	}
}

func limitsToU256(limits map[string]*big.Int) map[string]*uint256.Int {
	out := make(map[string]*uint256.Int, len(limits))
	for k, v := range limits {
		out[k] = bigToU256OrZero(v)
	}
	return out
}

// bigToU256OrZero converts a non-negative big.Int that fits 256 bits. A nil, negative, or oversized value
// becomes zero for amounts; limits saturate separately in ToU256Balances.
func bigToU256OrZero(b *big.Int) *uint256.Int {
	if b == nil || b.Sign() < 0 || b.BitLen() > 256 {
		return new(uint256.Int)
	}
	return uint256.MustFromBig(b)
}
