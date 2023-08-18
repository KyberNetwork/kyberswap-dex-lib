package balancercomposablestable

import "errors"

var (
	ErrorStableGetBalanceDidntConverge = errors.New("STABLE_GET_BALANCE_DIDNT_CONVERGE")
	ErrorInvalidAmountOutCalculated    = errors.New("INVALID_AMOUNT_OUT_CALCULATED")
)
