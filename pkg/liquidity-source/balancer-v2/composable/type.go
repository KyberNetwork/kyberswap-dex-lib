package composable

import (
	"github.com/holiman/uint256"
)

type LastJoinExitData struct {
	LastJoinExitAmplification *uint256.Int
	LastPostJoinExitInvariant *uint256.Int
}

type TokenRateCache struct {
	Rate     *uint256.Int
	OldRate  *uint256.Int
	Duration *uint256.Int
	Expires  *uint256.Int
}

type Gas struct {
	Swap int64
}
