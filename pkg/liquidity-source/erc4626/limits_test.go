package erc4626

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var limitCfg = VaultCfg{Gas: GasCfg{Deposit: 1}, DepositsPausedMethod: "depositsPaused", MaxSupplyMethod: "maxSupply"}

func TestResolveLimits(t *testing.T) {
	t.Parallel()
	s := PoolState{RawPaused: big.NewInt(1), MaxSupply: big.NewInt(100), TotalSupply: big.NewInt(70)}
	require.NoError(t, s.ResolveLimits(limitCfg))
	assert.True(t, s.Paused)
	assert.Equal(t, int64(30), s.MintHeadroom.Int64())

	// supply at or above the cap leaves zero headroom, not a negative or nil one
	s = PoolState{RawPaused: big.NewInt(0), MaxSupply: big.NewInt(100), TotalSupply: big.NewInt(100)}
	require.NoError(t, s.ResolveLimits(limitCfg))
	assert.False(t, s.Paused)
	require.NotNil(t, s.MintHeadroom)
	assert.Zero(t, s.MintHeadroom.Sign())
}

func TestResolveLimitsFailsOnMissingRead(t *testing.T) {
	t.Parallel()
	// a reverted pause or cap read must fail the refresh, not read as "not paused / no cap"
	s := PoolState{MaxSupply: big.NewInt(100), TotalSupply: big.NewInt(70)}
	assert.ErrorIs(t, s.ResolveLimits(limitCfg), ErrReadFailed)
	s = PoolState{RawPaused: big.NewInt(0), MaxSupply: big.NewInt(100)}
	assert.ErrorIs(t, s.ResolveLimits(limitCfg), ErrReadFailed)

	// not configured: nothing tracked
	s = PoolState{}
	require.NoError(t, s.ResolveLimits(VaultCfg{Gas: GasCfg{Deposit: 1}}))
	assert.False(t, s.Paused)
	assert.Nil(t, s.MintHeadroom)
}

func TestResolveMaxDeposit(t *testing.T) {
	t.Parallel()
	zero, gate, total := big.NewInt(0), big.NewInt(0xbeef), big.NewInt(561)
	for name, tc := range map[string]struct {
		s    PoolState
		want *big.Int
	}{
		"read failed":     {PoolState{}, total},
		"closed":          {PoolState{MaxDeposit: zero}, zero}, // e.g. Aave stata over its supply cap
		"ungated vaultV2": {PoolState{MaxDeposit: zero, ReceiveSharesGate: zero, SendAssetsGate: zero}, total},
		"gated vaultV2":   {PoolState{MaxDeposit: zero, ReceiveSharesGate: zero, SendAssetsGate: gate}, zero},
		"half read gates": {PoolState{MaxDeposit: zero, ReceiveSharesGate: zero}, zero},
		"bounded":         {PoolState{MaxDeposit: big.NewInt(7), ReceiveSharesGate: zero, SendAssetsGate: zero}, big.NewInt(7)},
		"no limit":        {PoolState{MaxDeposit: new(big.Int).Lsh(big.NewInt(1), 200)}, nil},
	} {
		tc.s.TotalAssets = total
		tc.s.ResolveMaxDeposit()
		assert.Equal(t, tc.want, tc.s.MaxDeposit, name)
	}
}
