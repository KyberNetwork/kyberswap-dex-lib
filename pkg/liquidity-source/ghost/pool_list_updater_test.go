package ghost

import (
	"context"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetNewPools(t *testing.T) {
	t.Parallel()

	updater := NewPoolsListUpdater(&Config{DexID: DexType, PoolPath: "pools/ethereum.json"}, nil)

	pools, metadata, err := updater.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	assert.Nil(t, metadata)
	require.Len(t, pools, 1)

	p := pools[0]
	assert.Equal(t, "ghost_0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48_0xdac17f958d2ee523a2206206994597c13d831ec7", p.Address)
	assert.Equal(t, DexType, p.Type)
	assert.Equal(t, DexType, p.Exchange)
	require.Len(t, p.Tokens, 2)
	assert.Equal(t, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", p.Tokens[0].Address)
	assert.True(t, p.Tokens[0].Swappable)
	require.Len(t, p.Reserves, 2)
	assert.Equal(t, defaultReserves, p.Reserves[0])
	assert.Equal(t, "{}", p.Extra)

	var se StaticExtra
	require.NoError(t, unmarshalStaticExtra(p.StaticExtra, &se))
	assert.Equal(t, "0xA9C9a8FB36Ce3e5ffBAC3757dA7141262723541F", se.ZeroToOne.SourceRouter)
	assert.Equal(t, uint32(1), se.ZeroToOne.LocalDomain)
	assert.Equal(t, "0xeB1b48b238E15A62e1858a601B6BfFdf41163AE3", se.OneToZero.SourceRouter)

	// Second call returns nothing further; the updater only initializes pools once.
	pools, _, err = updater.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	assert.Nil(t, pools)
}

func TestGetNewPools_MisconfiguredPoolPath(t *testing.T) {
	t.Parallel()

	updater := NewPoolsListUpdater(&Config{DexID: DexType, PoolPath: "pools/does-not-exist.json"}, nil)

	pools, _, err := updater.GetNewPools(context.Background(), nil)
	assert.Error(t, err)
	assert.Nil(t, pools)
}

func unmarshalStaticExtra(raw string, se *StaticExtra) error {
	return json.Unmarshal([]byte(raw), se)
}
