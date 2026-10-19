package hyperamm

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeHyperAMM

	defaultGas = int64(536883)

	// MaxAge bounds the age of the tracked fair-price snapshot under StaleCheck. The tracker stamps
	// Timestamp on every successful refresh, so an older pool is either drifting from the live oracle
	// or failing to refresh at all, e.g. while the fee module's oracle reverts (Pyth StalePrice), in
	// which case every swap reverts too.
	MaxAge = time.Minute
)

var (
	Router     = "0x18ebC3F95CD6B5db55A15864079019C5d2b83DBC"
	RouterAddr = common.HexToAddress(Router)

	ErrPoolPaused          = errors.WithMessage(pool.ErrUnsupported, "hyperamm: pool is paused")
	ErrInvalidToken        = errors.New("hyperamm: invalid token")
	ErrZeroAmountIn        = errors.New("hyperamm: zero amount in")
	ErrZeroAmountOut       = errors.New("hyperamm: zero amount out")
	ErrInsufficientReserve = errors.New("hyperamm: insufficient reserve")
	ErrZeroFairPrice       = errors.New("hyperamm: zero fair price")
	ErrStale               = errors.New("hyperamm: pool state is stale")
)
