package nuriv2

import (
	"math/big"

	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	DexType              = "nuri-v2"
	graphSkipLimit       = 5000
	graphFirstLimit      = 1000
	defaultTokenDecimals = 18
	zeroString           = "0"
	emptyString          = ""
	tickChunkSize        = 100
)

const (
	methodGetLiquidity = "liquidity"
	methodGetSlot0     = "slot0"
	methodCurrentFee   = "currentFee"
	methodTickSpacing  = "tickSpacing"
	methodTicks        = "ticks"
)

var (
	zeroBI = big.NewInt(0)
	// Waiting the SC team to estimate the CrossInitTickGas at thread:
	// https://team-kyber.slack.com/archives/C05V8NL8CSF/p1702621669962399.
	// For now, keep the BaseGas = 125000 (as the previous config), CrossInitTickGas = 0.
	defaultGas = Gas{BaseGas: 125000, CrossInitTickGas: 0}
)

var (
	ErrTickNil      = errors.WithMessage(pool.ErrUnsupported, "tick is nil")
	ErrV3TicksEmpty = errors.WithMessage(pool.ErrUnsupported, "v3Ticks empty")
)
