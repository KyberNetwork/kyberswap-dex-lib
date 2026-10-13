package tickerdualpool

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestRawInventoryAndManagerCashMoveTogether(t *testing.T) {
	// Pool owns 900 cash and 100 old claims. Deploy 1,000; teardown owes
	// PoolManager 200 net. Old claims are first exchanged for physical cash.
	raw, claims, err := inventoryAfterJIT(uint256.NewInt(900), uint256.NewInt(100), uint256.NewInt(1000), uint256.NewInt(800))
	require.NoError(t, err)
	require.Equal(t, "800", raw.Dec())
	require.True(t, claims.IsZero())
	manager, err := managerCashAfterJIT(uint256.NewInt(100), uint256.NewInt(100), uint256.NewInt(1000), uint256.NewInt(800))
	require.NoError(t, err)
	require.Equal(t, "200", manager.Dec())
	// A positive JIT delta mints a claim without adding cash to PoolManager.
	raw, claims, err = inventoryAfterJIT(uint256.NewInt(900), uint256.NewInt(100), uint256.NewInt(1000), uint256.NewInt(1100))
	require.NoError(t, err)
	require.Equal(t, "1000", raw.Dec())
	require.Equal(t, "100", claims.Dec())
	manager, err = managerCashAfterJIT(uint256.NewInt(100), uint256.NewInt(100), uint256.NewInt(1000), uint256.NewInt(1100))
	require.NoError(t, err)
	require.True(t, manager.IsZero())
	require.Equal(t, "100", unbackedClaims(claims, manager).Dec())
	// Cash belonging to other users can already back the new claims.
	require.True(t, unbackedClaims(claims, uint256.NewInt(100)).IsZero())
	require.Equal(t, "1", unbackedClaims(claims, uint256.NewInt(99)).Dec())
}

func TestUnpaidClaimsCannotBeRedeemed(t *testing.T) {
	_, err := managerCashAfterJIT(uint256.NewInt(99), uint256.NewInt(100), uint256.NewInt(1000), uint256.NewInt(800))
	require.ErrorIs(t, err, ErrVaultIlliquid)
}
