package cloberob

import (
	"errors"
)

const (
	DexType = "clober-ob"

	bookManagerMethodGetDepth   = "getDepth"
	bookManagerMethodGetHighest = "getHighest"

	bookViewerMethodGetLiquidity = "getLiquidity"

	int24Min = -(1 << 23)

	maxTickLimit = 100

	defaultTakeGas int64 = 59197
	defaultBaseGas int64 = 57713
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrNoLiquidity  = errors.New("no liquidity")
)
