package limitorder

import "errors"

var ErrCannotFulfillAmountIn = errors.New("cannot fulfill amountIn")
var InvalidSwapInfo = errors.New("invalid swap info")
