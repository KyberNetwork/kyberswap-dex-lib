package efficientprop

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// The directory, the DexType and the exchange constant share one name, the source is a PropAMM one, and its
// listings share their pool's custody as a swap limit.
func TestRegistration(t *testing.T) {
	assert.Equal(t, valueobject.ExchangeEfficientProp, DexType)
	assert.True(t, valueobject.IsPropAMMSource(DexType))
	assert.True(t, pool.UsesSwapLimit(DexType, DexType))
	assert.True(t, pool.UsesSwapLimit("efficient-prop-custom", DexType), "a custom dex id keeps the pool type's limit")
}
