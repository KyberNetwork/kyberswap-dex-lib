package pamm

import (
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/kipseli"
)

const (
	DexType = "kipseli-pamm"

	// sampleSize: count of power-of-10 amountIn levels per direction.
	sampleSize = 15

	// Offchain TTL for a Titan/PUR snapshot. On-chain reads remain exact:
	// Kipseli's target calls PUR.getState(..., block.timestamp, block.timestamp).
	priorityUpdateFreshnessTTL = 30 * time.Second

	priorityUpdateLaneIndex = 0

	// positionCapMethod reads the venue's maximum position for a token.
	positionCapMethod = "caps"
)

// maxInSampleBps: fractions of vault reserve to probe for tighter interpolation near capacity.
var maxInSampleBps = []int{
	200, 500, 1000, 1500, 2200, 3200, 4000,
	4500, 5000, 5600, 6200, 6800,
	7300, 7900, 8500, 9100, 9900,
}

var (
	ErrInvalidToken          = kipseli.ErrInvalidToken
	ErrInsufficientLiquidity = kipseli.ErrInsufficientLiquidity
)
