package weighted

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// A negative legacy amount must be rejected: SetFromBig alone would wrap it to about 2^256.
func TestTokenAmtFromBigRejectsNegative(t *testing.T) {
	_, err := tokenAmtFromBigWeighted(&pool.TokenAmount{Amount: big.NewInt(-1)})
	require.Error(t, err)
	out, err := tokenAmtFromBigWeighted(&pool.TokenAmount{Amount: big.NewInt(5)})
	require.NoError(t, err)
	require.Equal(t, uint64(5), out.Amount.Uint64())
}
