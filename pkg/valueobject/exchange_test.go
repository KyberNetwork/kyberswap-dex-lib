package valueobject

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Verify the TrebleSwap exchange identifier is stable.
func TestExchangeTrebleSwap(t *testing.T) {
	t.Parallel()

	require.Equal(t, "trebleswap", ExchangeTrebleSwap)
}
