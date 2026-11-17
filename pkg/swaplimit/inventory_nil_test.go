package swaplimit

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// Inventory reads its deltas, so a nil delta must be an error, not a panic (order-book sends one).
func TestInventoryUpdateLimitNilDeltaIsError(t *testing.T) {
	inv := NewInventory("x", map[string]*uint256.Int{"a": uint256.NewInt(100)})
	require.NotPanics(t, func() {
		_, _, err := inv.UpdateLimit("a", "b", nil, uint256.NewInt(0))
		require.Error(t, err)
	})
}
