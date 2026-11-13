package pool

import (
	"github.com/holiman/uint256"
)

// TokenAmt is TokenAmount with a uint256 amount.
type TokenAmt struct {
	Token     string
	Amount    uint256.Int
	AmountUsd float64
}

// CalcAmtOutParams is CalcAmountOutParams with a uint256 input amount.
type CalcAmtOutParams struct {
	TokenAmountIn TokenAmt
	TokenOut      string
	Limit         SwapLimit
}

// CalcAmtOutResult is CalcAmountOutResult with uint256 amounts.
type CalcAmtOutResult struct {
	TokenAmountOut         *TokenAmt
	Fee                    *TokenAmt
	RemainingTokenAmountIn *TokenAmt
	Gas                    int64
	SwapInfo               any
}

// CalcAmtInParams is CalcAmountInParams with a uint256 output amount.
type CalcAmtInParams struct {
	TokenAmountOut TokenAmt
	TokenIn        string
	Limit          SwapLimit
}

// CalcAmtInResult is CalcAmountInResult with uint256 amounts.
type CalcAmtInResult struct {
	TokenAmountIn           *TokenAmt
	RemainingTokenAmountOut *TokenAmt
	Fee                     *TokenAmt
	Gas                     int64
	SwapInfo                any
}

// ICalcAmtOut is implemented by simulators that calculate amountOut natively in uint256.
// Callers should type-assert and fall back to IPoolSimulator.CalcAmountOut when absent.
type ICalcAmtOut interface {
	CalcAmtOut(params CalcAmtOutParams) (*CalcAmtOutResult, error)
}

// ICalcAmtIn is implemented by simulators that calculate amountIn natively in uint256.
// Callers should type-assert and fall back to IPoolExactOutSimulator.CalcAmountIn when absent.
type ICalcAmtIn interface {
	CalcAmtIn(params CalcAmtInParams) (*CalcAmtInResult, error)
}

// IsValid mirrors CalcAmountOutResult.IsValid. uint256 amounts cannot be negative, so the remaining-amount
// check reduces to a nil check.
func (r *CalcAmtOutResult) IsValid() bool {
	return r.TokenAmountOut != nil && !r.TokenAmountOut.Amount.IsZero()
}

// UpdateStateParams is UpdateBalanceParams with uint256 amounts. Fee is a value, so a zero fee needs no nil check.
type UpdateStateParams struct {
	TokenAmountIn  TokenAmt
	TokenAmountOut TokenAmt
	Fee            TokenAmt
	SwapInfo       any
	SwapLimit      SwapLimit
}

// IUpdateState is implemented by simulators that update their state from uint256 amounts.
// Callers should type-assert and fall back to IPoolSimulator.UpdateBalance when absent.
type IUpdateState interface {
	UpdateState(params UpdateStateParams)
}
