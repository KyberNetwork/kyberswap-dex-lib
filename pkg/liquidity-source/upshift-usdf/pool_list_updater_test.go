package upshiftusdf

import (
	"context"
	"testing"

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
	assert.Equal(t, Vault, p.Address)
	assert.Equal(t, DexType, p.Exchange)
	assert.Equal(t, DexType, p.Type)
	require.Len(t, p.Tokens, 2)
	assert.Equal(t, USDC, p.Tokens[0].Address)
	assert.Equal(t, USDF, p.Tokens[1].Address)
	assert.Equal(t, entity.PoolReserves{"0", "0"}, p.Reserves)

	// placeholder state must be loadable by the simulator
	_, err = NewPoolSimulator(p)
	require.NoError(t, err)

	pools, _, err = u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, pools)
}
