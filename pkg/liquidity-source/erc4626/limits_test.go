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
