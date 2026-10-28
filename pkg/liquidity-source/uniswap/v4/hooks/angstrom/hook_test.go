package angstrom

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Attestation age is off-chain freshness: indexing (StaleCheck=false) must not reject old attestations.
func TestBeforeSwapStaleCheck(t *testing.T) {
	old := HookExtra{LatestAttestationsTime: time.Now().Add(-time.Hour).Unix(), UnlockedFee: big.NewInt(100)}
	for _, staleCheck := range []bool{false, true} {
		h := &Hook{extra: old, staleCheck: staleCheck}
		_, err := h.BeforeSwap(nil)
		if staleCheck {
			assert.ErrorIs(t, err, ErrOutdatedAttestations)
		} else {
			require.NoError(t, err)
		}
	}
}
