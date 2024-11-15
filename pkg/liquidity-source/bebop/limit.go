package bebop

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
)

// NewLimit creates a new SingleSwapLimit.
// Deprecated: directly use swaplimit.NewSingleSwapLimit.
func NewLimit(_ map[string]*big.Int) pool.SwapLimit {
	return swaplimit.NewSingleSwapLimit(DexType)
}
