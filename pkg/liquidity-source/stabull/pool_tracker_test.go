package stabull

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestPoolTracker_FetchPoolStateFromNode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name        string
		rpcURL      string
		poolAddress string
	}{
		{
			name:        "Polygon - NZDS/USDC Pool",
			rpcURL:      "https://polygon-mainnet.g.alchemy.com/v2/IqvzEgP3ce5i1ruu_uNyK",
			poolAddress: "0xdcb7efACa996fe2985138bF31b647EFcd1D0901a",
		},
		{
			name:        "Base - BRZ/USDC Pool",
			rpcURL:      "https://base-mainnet.g.alchemy.com/v2/IqvzEgP3ce5i1ruu_uNyK",
			poolAddress: "0x8A908aE045E611307755A91f4D6ECD04Ed31EB1B",
		},
		{
			name:        "Ethereum - NZDS/USDC Pool",
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
				DexID: "stabull-test",
			}

			tracker, err := NewPoolTracker(config, client)
			require.NoError(t, err)
			require.NotNil(t, tracker)

			// Execute
			ctx := context.Background()
			reserves, extra, err := tracker.fetchPoolStateFromNode(ctx, tt.poolAddress)

			// Assert
			require.NoError(t, err, "Should successfully fetch pool state")
			require.NotNil(t, reserves, "Reserves should not be nil")
			require.Len(t, reserves, 2, "Should have 2 reserves")

			t.Logf("=== Pool State ===")
			t.Logf("Pool: %s", tt.poolAddress)
			t.Logf("\nReserves:")
			t.Logf("  Reserve 0: %s", reserves[0].String())
			t.Logf("  Reserve 1: %s", reserves[1].String())

			// Validate reserves are non-zero
			assert.True(t, reserves[0].Cmp(big.NewInt(0)) > 0, "Reserve 0 should be positive")
			assert.True(t, reserves[1].Cmp(big.NewInt(0)) > 0, "Reserve 1 should be positive")

			// Validate curve parameters
			t.Logf("\nCurve Parameters (Greeks):")
			t.Logf("  Alpha (α): %s", extra.CurveParams.Alpha)
			t.Logf("  Beta (β): %s", extra.CurveParams.Beta)
			t.Logf("  Delta (δ): %s", extra.CurveParams.Delta)
			t.Logf("  Epsilon (ε): %s", extra.CurveParams.Epsilon)
			t.Logf("  Lambda (λ): %s", extra.CurveParams.Lambda)

			assert.NotEmpty(t, extra.CurveParams.Alpha, "Alpha should not be empty")
			assert.NotEmpty(t, extra.CurveParams.Beta, "Beta should not be empty")
			assert.NotEmpty(t, extra.CurveParams.Delta, "Delta should not be empty")
			assert.NotEmpty(t, extra.CurveParams.Epsilon, "Epsilon should not be empty")
			assert.NotEmpty(t, extra.CurveParams.Lambda, "Lambda should not be empty")

			// Parse and validate parameter values are reasonable
			alpha, ok := new(big.Int).SetString(extra.CurveParams.Alpha, 10)
			assert.True(t, ok && alpha.Cmp(big.NewInt(0)) > 0, "Alpha should be positive")

			beta, ok := new(big.Int).SetString(extra.CurveParams.Beta, 10)
			assert.True(t, ok && beta.Cmp(big.NewInt(0)) > 0, "Beta should be positive")

			delta, ok := new(big.Int).SetString(extra.CurveParams.Delta, 10)
			assert.True(t, ok && delta.Cmp(big.NewInt(0)) > 0, "Delta should be positive")

			epsilon, ok := new(big.Int).SetString(extra.CurveParams.Epsilon, 10)
			assert.True(t, ok && epsilon.Cmp(big.NewInt(0)) > 0, "Epsilon should be positive")

			lambda, ok := new(big.Int).SetString(extra.CurveParams.Lambda, 10)
			assert.True(t, ok && lambda.Cmp(big.NewInt(0)) > 0, "Lambda should be positive")

			// Log oracle info if available
			if extra.BaseOracleAddress != "" {
				t.Logf("\nOracle Information:")
				t.Logf("  Base Oracle: %s", extra.BaseOracleAddress)
				if extra.BaseOracleRate != "" {
					t.Logf("  Base Rate: %s", extra.BaseOracleRate)
				}
			}
			if extra.QuoteOracleAddress != "" {
				t.Logf("  Quote Oracle: %s", extra.QuoteOracleAddress)
				if extra.QuoteOracleRate != "" {
					t.Logf("  Quote Rate: %s", extra.QuoteOracleRate)
				}
			}
			if extra.OracleRate != "" {
				t.Logf("  Derived Oracle Rate: %s", extra.OracleRate)
			}
		})
	}
}

func TestPoolTracker_GetNewPoolState(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name        string
		rpcURL      string
		poolAddress string
		token0      string
		token1      string
	}{
		{
			name:        "Polygon - NZDS/USDC",
			rpcURL:      "https://polygon-mainnet.g.alchemy.com/v2/IqvzEgP3ce5i1ruu_uNyK",
			poolAddress: "0xdcb7efACa996fe2985138bF31b647EFcd1D0901a",
			token0:      "0xFbBE4b730e1e77d02dC40fEdF94382802eab3B5",  // NZDS
			token1:      "0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359", // USDC
		},
		{
			name:        "Base - BRZ/USDC",
			rpcURL:      "https://base-mainnet.g.alchemy.com/v2/IqvzEgP3ce5i1ruu_uNyK",
			poolAddress: "0x8A908aE045E611307755A91f4D6ECD04Ed31EB1B",
			token0:      "0xE9185Ee218cae427aF7B9764A011bb89FeA76144", // BRZ
			token1:      "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913", // USDC
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			client := ethrpc.New(tt.rpcURL)
			require.NotNil(t, client)

			// Set multicall contract address
			client.SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

			config := &Config{
				DexID: "stabull-test",
			}

			tracker, err := NewPoolTracker(config, client)
			require.NoError(t, err)

			// Create initial pool
			poolEntity := entity.Pool{
				Address:  tt.poolAddress,
				Exchange: "stabull",
				Type:     DexType,
				Tokens: []*entity.PoolToken{
					{Address: tt.token0, Decimals: 18},
					{Address: tt.token1, Decimals: 6},
				},
				Reserves: []string{"1000000000000000000", "1000000"},
			}

			// Execute
			ctx := context.Background()
			updatedPool, err := tracker.GetNewPoolState(ctx, poolEntity, pool.GetNewPoolStateParams{})

			// Assert
			require.NoError(t, err)
			assert.NotEmpty(t, updatedPool.Reserves)
			assert.Len(t, updatedPool.Reserves, 2)

			t.Logf("Updated Pool State:")
			t.Logf("  Reserve 0: %s", updatedPool.Reserves[0])
			t.Logf("  Reserve 1: %s", updatedPool.Reserves[1])

			// Validate extra data
			var extra Extra
			err = json.Unmarshal([]byte(updatedPool.Extra), &extra)
			require.NoError(t, err)

			t.Logf("  Greeks:")
			t.Logf("    Alpha: %s", extra.CurveParams.Alpha)
			t.Logf("    Beta: %s", extra.CurveParams.Beta)
			t.Logf("    Delta: %s", extra.CurveParams.Delta)
			t.Logf("    Epsilon: %s", extra.CurveParams.Epsilon)
			t.Logf("    Lambda: %s", extra.CurveParams.Lambda)
		})
	}
}
