package liquid

import (
	"context"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestGetNewPools_ByChain(t *testing.T) {
	// The Ethereum yaml has no chainId, so zero must keep the three LiquidRefer vaults.
	eth, _, err := NewPoolListUpdater(&Config{}, nil).GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, eth, 3)
	for _, p := range eth {
		var se StaticExtra
		require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &se))
		require.Equal(t, liquidReferAddress, se.LiquidRefer)
		require.False(t, se.SimpleTeller)
		require.Equal(t, valueobject.ExchangeEtherfiLiquid, p.Exchange)
	}

	hl, _, err := NewPoolListUpdater(&Config{ChainID: valueobject.ChainIDHyperEVM}, nil).GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, hl, 1)
	require.Equal(t, lhype, hl[0].Address)
	require.Equal(t, valueobject.ExchangeLoopLiquid, hl[0].Exchange)
	require.Equal(t, whype, hl[0].Tokens[1].Address)
	require.Equal(t, sthype, hl[0].Tokens[2].Address)
	var se StaticExtra
	require.NoError(t, json.Unmarshal([]byte(hl[0].StaticExtra), &se))
	require.True(t, se.SimpleTeller)
	require.Equal(t, hexAddr("0xfd83c1ca0c04e096d129275126fade1dc45bf4f0"), se.Teller)
}
