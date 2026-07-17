package liquidcore

import (
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeLiquidCore

	defaultGas = 139441

	// MaxAge bounds staleness of the sampled quote ladder: liquidcore's fee is
	// derived from a live oracle, so a ladder sampled too long ago can drift
	// off the contract's actual quote curve.
	MaxAge = time.Minute
)
