package stabull

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPoolsListUpdater_GetNewPool_SpecificAddress tests fetching a specific pool by address
func TestPoolsListUpdater_GetNewPool_SpecificAddress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name        string
		chainID     uint
		rpcURL      string
		poolAddress string
	}{
		{
			name:        "Polygon - NZDS/USDC Pool",
			chainID:     137,
			rpcURL:      "https://polygon-mainnet.g.alchemy.com/v2/IqvzEgP3ce5i1ruu_uNyK",
			poolAddress: "0xdcb7efACa996fe2985138bF31b647EFcd1D0901a",
		},
		{
			name:        "Base - BRZ/USDC Pool",
			chainID:     8453,
			rpcURL:      "https://base-mainnet.g.alchemy.com/v2/IqvzEgP3ce5i1ruu_uNyK",
			poolAddress: "0x8A908aE045E611307755A91f4D6ECD04Ed31EB1B",
		},
		{
			name:        "Ethereum - NZDS/USDC Pool",
			chainID:     1,
			rpcURL:      "https://eth-mainnet.g.alchemy.com/v2/IqvzEgP3ce5i1ruu_uNyK",
			poolAddress: "0xe37D763c7c4cdd9A8f085F7DB70139a0843529F3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			client := ethrpc.New(tt.rpcURL)
			client.SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
			require.NotNil(t, client)

			config := &Config{
				DexID:          "stabull-test",
				FactoryAddress: "", // Not needed for direct pool fetch
			}

			updater := NewPoolsListUpdater(config, client)

			// Execute
			ctx := context.Background()
			pool, err := updater.getNewPool(ctx, tt.poolAddress)

			// Assert
			require.NoError(t, err, "Should successfully fetch pool")
			require.NotNil(t, pool, "Pool should not be nil")

			t.Logf("Pool Address: %s", pool.Address)
			t.Logf("Type: %s", pool.Type)
			t.Logf("Exchange: %s", pool.Exchange)

			assert.Equal(t, DexType, pool.Type)
			assert.Equal(t, 2, len(pool.Tokens))
			assert.Equal(t, 2, len(pool.Reserves))

			// Log token info
			for i, token := range pool.Tokens {
				t.Logf("Token %d: %s (decimals: %d)", i, token.Address, token.Decimals)
			}

			// Log reserves
			for i, reserve := range pool.Reserves {
				t.Logf("Reserve %d: %s", i, reserve)
			}

			// Validate and log extra data
			var extra Extra
			err = json.Unmarshal([]byte(pool.Extra), &extra)
			require.NoError(t, err)

			t.Logf("Curve Parameters:")
			t.Logf("  Alpha: %s", extra.CurveParams.Alpha)
			t.Logf("  Beta: %s", extra.CurveParams.Beta)
			t.Logf("  Delta: %s", extra.CurveParams.Delta)
			t.Logf("  Epsilon: %s", extra.CurveParams.Epsilon)
			t.Logf("  Lambda: %s", extra.CurveParams.Lambda)

			if extra.BaseOracleAddress != "" {
				t.Logf("Base Oracle: %s", extra.BaseOracleAddress)
			}
			if extra.QuoteOracleAddress != "" {
				t.Logf("Quote Oracle: %s", extra.QuoteOracleAddress)
			}
		})
	}
}
