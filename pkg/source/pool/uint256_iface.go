package pool

import (
	"github.com/holiman/uint256"
)

// TokenAmountU is TokenAmount with a uint256 amount.
type TokenAmountU struct {
	Token     string
	Amount    uint256.Int
	AmountUsd float64
}

// CalcAmountOutParamsU is CalcAmountOutParams with a uint256 input amount.
type CalcAmountOutParamsU struct {
	TokenAmountIn TokenAmountU
	TokenOut      string
	Limit         SwapLimit
}

// CalcAmountOutResultU is CalcAmountOutResult with uint256 amounts.
type CalcAmountOutResultU struct {
	TokenAmountOut         *TokenAmountU
	Fee                    *TokenAmountU
	RemainingTokenAmountIn *TokenAmountU
	Gas                    int64
	SwapInfo               any
}

// CalcAmountInParamsU is CalcAmountInParams with a uint256 output amount.
type CalcAmountInParamsU struct {
	TokenAmountOut TokenAmountU
	TokenIn        string
	Limit          SwapLimit
}

// CalcAmountInResultU is CalcAmountInResult with uint256 amounts.
type CalcAmountInResultU struct {
	TokenAmountIn           *TokenAmountU
	RemainingTokenAmountOut *TokenAmountU
	Fee                     *TokenAmountU
	Gas                     int64
	SwapInfo                any
}

// ICalcAmtOut is implemented by simulators that calculate amountOut natively in uint256.
// Callers should type-assert and fall back to IPoolSimulator.CalcAmountOut when absent.
type ICalcAmtOut interface {
	CalcAmtOut(params CalcAmountOutParamsU) (*CalcAmountOutResultU, error)
}

// ICalcAmtIn is implemented by simulators that calculate amountIn natively in uint256.
// Callers should type-assert and fall back to IPoolExactOutSimulator.CalcAmountIn when absent.
type ICalcAmtIn interface {
	CalcAmtIn(params CalcAmountInParamsU) (*CalcAmountInResultU, error)
}
