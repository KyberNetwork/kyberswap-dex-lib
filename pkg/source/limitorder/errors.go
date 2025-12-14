package limitorder

import "errors"

var ErrCannotFulfillAmountIn = errors.New("cannot fulfill amountIn")
var ErrCannotFulfillAmountOut = errors.New("cannot fulfill amountOut")
var ErrInvalidSwapInfo = errors.New("invalid swap info")
var ErrSameSenderMaker = errors.New("swap recipient is the same as order receiver")
var ErrGetOpSignaturesFailed = errors.New("failed to get operator signatures")
