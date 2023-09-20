package wombatlsd

import "errors"

var (
	ErrTheSameAddress          = errors.New("tokenIn and tokenOut has the same address")
	ErrFromAmountIsZero        = errors.New("fromAmount equals zero")
	ErrAssetIsNotExist         = errors.New("asset is not exist")
	ErrCashNotEnough           = errors.New("cash is not enough")
	ErrCoreUnderflow           = errors.New("core underflow")
	ErrCovRatioLimitExceeded   = errors.New("cov ratio limit exceeded")
	ErrWombatAssetAlreadyPause = errors.New("wombat asset already paused")
)
