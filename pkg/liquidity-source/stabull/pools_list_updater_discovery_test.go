package stabull

import (
	"context"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// TestPoolsListUpdater_GetNewPools tests the complete pool discovery flow
func TestPoolsListUpdater_GetNewPools(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name           string
		chainID        uint
		rpcURL         string
		factoryAddress string
		expectedMin    int // Minimum number of pools expected
	}{
		{
			name:           "Polygon Pool Discovery",
			chainID:        137,
			rpcURL:         "https://polygon-mainnet.g.alchemy.com/v2/IqvzEgP3ce5i1ruu_uNyK",
			factoryAddress: "0x3c60234db40e6e5b57504e401b1cdc79d91faf89",
			expectedMin:    13, // We know there are 13 pools on Polygon
		},
		{
			name:           "Base Pool Discovery",
			chainID:        8453,
			rpcURL:         "https://base-mainnet.g.alchemy.com/v2/IqvzEgP3ce5i1ruu_uNyK",
			factoryAddress: "0x86Ba17ebf8819f7fd32Cf1A43AbCaAe541A5BEbf",
			expectedMin:    8, // We know there are 8 pools on Base
		},
		{
			name:           "Ethereum Pool Discovery",
			chainID:        1,
			rpcURL:         "https://eth-mainnet.g.alchemy.com/v2/IqvzEgP3ce5i1ruu_uNyK",
			factoryAddress: "0x2e9E34b5Af24b66F12721113C1C8FFcbB7Bc8051",
			expectedMin:    5, // We know there are 5 pools on Ethereum
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
				ChainID:        tt.chainID,
				FactoryAddress: tt.factoryAddress,
				NewPoolLimit:   100,
			}

			updater := NewPoolsListUpdater(config, client)

			// Execute
			ctx := context.Background()
			pools, metadata, err := updater.GetNewPools(ctx, nil)

			// Assert
			require.NoError(t, err, "Should successfully discover pools")
			require.NotNil(t, pools, "Pools should not be nil")
			require.GreaterOrEqual(t, len(pools), tt.expectedMin, "Should discover at least the expected number of pools")

			t.Logf("Discovered %d pools on chain %d", len(pools), tt.chainID)

			// Log each pool
			for i, pool := range pools {
				t.Logf("  Pool %d: %s", i+1, pool.Address)
				t.Logf("    Token0: %s (%d decimals)", pool.Tokens[0].Address, pool.Tokens[0].Decimals)
				t.Logf("    Token1: %s (%d decimals)", pool.Tokens[1].Address, pool.Tokens[1].Decimals)
				t.Logf("    Reserve0: %s", pool.Reserves[0])
				t.Logf("    Reserve1: %s", pool.Reserves[1])
			}

			// Metadata: Since we pass nil as metadataBytes, it should remain nil or be empty
			// The factory-based discovery doesn't use offset-based pagination
			t.Logf("Metadata type: %T, value: %v", metadata, metadata)
		})
	}
}
