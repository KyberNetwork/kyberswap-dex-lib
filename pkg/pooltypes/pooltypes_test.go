package pooltypes

import (
	"testing"

	"github.com/stretchr/testify/require"

	algebraintegral "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/algebra/integral"
)

// Ensure TrebleSwap is wired to the algebra-integral pool type.
func TestTrebleSwapPoolType(t *testing.T) {
	t.Parallel()

	require.Equal(t, algebraintegral.DexType, PoolTypes.TrebleSwap, "TrebleSwap must reuse algebra-integral pool type")
}
