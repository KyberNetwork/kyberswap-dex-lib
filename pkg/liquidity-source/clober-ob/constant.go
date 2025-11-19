package cloberob

import (
	"errors"
)

const (
	DexType = "clober-ob"

	bookManagerMethodGetHighest = "getHighest"
	bookManagerMethodGetDepth   = "getDepth"

	bookViewerMethodGetLiquidity      = "getLiquidity"
	bookViewerMethodGetExpectedOutput = "getExpectedOutput"

	int24Min = -(1 << 23)

	maxTickLimit = 1000

	defaultGas = 0
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrNoLiquidity  = errors.New("no liquidity")
	ErrInvalidState = errors.New("invalid state")
)
