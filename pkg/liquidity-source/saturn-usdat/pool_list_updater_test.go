package saturnusdat

import (
	"context"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

func TestGetNewPools(t *testing.T) {
	t.Parallel()
	u := NewPoolsListUpdater(&Config{DexID: DexType}, nil)

	pools, cursor, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	assert.Nil(t, cursor)
	require.Len(t, pools, 1)

	p := pools[0]
	assert.Equal(t, USDat, p.Address)
	assert.Equal(t, DexType, p.Exchange)
	assert.Equal(t, DexType, p.Type)
	require.Len(t, p.Tokens, 2)
	assert.Equal(t, USDC, p.Tokens[0].Address)
	assert.Equal(t, USDat, p.Tokens[1].Address)
	assert.True(t, p.Tokens[0].Swappable && p.Tokens[1].Swappable)
	assert.Equal(t, entity.PoolReserves{"0", "0"}, p.Reserves)

	var static StaticExtra
	require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &static))
	assert.Equal(t, StaticExtra{Facility: Facility, Asset: USDC}, static)

	// placeholder state must be loadable by the simulator
	_, err = NewPoolSimulator(p)
	require.NoError(t, err)

	// only emitted once
	pools, _, err = u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, pools)
}
