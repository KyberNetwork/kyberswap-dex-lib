package limitorder

import "errors"

var ErrCannotFulfillAmountIn = errors.New("cannot fulfill amountIn")
var ErrCannotFulfillAmountOut = errors.New("cannot fulfill amountOut")
var InvalidSwapInfo = errors.New("invalid swap info")
var ErrSameSenderMaker = errors.New("swap recipient is the same as order receiver")
