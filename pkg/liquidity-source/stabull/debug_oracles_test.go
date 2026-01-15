package stabull

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/stretchr/testify/require"
)

func TestDebugOracleAddresses(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	rpcURL := os.Getenv("STABULL_RPC_ETHEREUM")
	if rpcURL == "" {
		rpcURL = "https://eth-mainnet.g.alchemy.com/v2/IqvzEgP3ce5i1ruu_uNyK"
	}

	client := ethrpc.New(rpcURL)
	require.NotNil(t, client)

	ctx := context.Background()
	updater := NewPoolsListUpdater(&Config{
		DexID:          "stabull-debug",
		ChainID:        1,
		FactoryAddress: "0x2e9E34b5Af24b66F12721113C1C8FFcbB7Bc8051",
		FromBlock:      0, // Start from beginning
	}, client)

	pools, _, err := updater.GetNewPools(ctx, nil)
	t.Logf("Discovery returned %d pools, error: %v", len(pools), err)
	require.NoError(t, err)
	require.NotEmpty(t, pools)

	for _, p := range pools {
		var extra Extra
		err := json.Unmarshal([]byte(p.Extra), &extra)
		require.NoError(t, err)

		// Print info for all pools to see if TRYB/GYEN exist
		t.Logf("Found pool: %s with tokens: %v", p.Address, p.Tokens)

		// Print info for TRYB and GYEN pools
		if len(p.Tokens) >= 2 {
			if p.Tokens[0].Symbol == "TRYB" || p.Tokens[0].Symbol == "GYEN" {
				t.Logf("\n=== %s Pool ===", p.Tokens[0].Symbol)
				t.Logf("Pool Address: %s", p.Address)
				t.Logf("Token0: %s (%s)", p.Tokens[0].Symbol, p.Tokens[0].Address)
				t.Logf("Token1: %s (%s)", p.Tokens[1].Symbol, p.Tokens[1].Address)
				t.Logf("BaseOracleAddress: '%s'", extra.BaseOracleAddress)
				t.Logf("QuoteOracleAddress: '%s'", extra.QuoteOracleAddress)
				t.Logf("BaseOracleRate: '%s'", extra.BaseOracleRate)
				t.Logf("QuoteOracleRate: '%s'", extra.QuoteOracleRate)
			}
		}
	}

	// Require that we found at least some pools
	require.NotEmpty(t, pools, "Should have found some pools")
}
