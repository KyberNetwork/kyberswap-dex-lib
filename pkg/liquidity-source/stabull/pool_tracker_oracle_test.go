package stabull

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// TestPoolTracker_FetchOracleRates tests fetching oracle rates
func TestPoolTracker_FetchOracleRates(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	client := ethrpc.New("https://polygon-mainnet.g.alchemy.com/v2/IqvzEgP3ce5i1ruu_uNyK")
	client.SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	require.NotNil(t, client)

	config := &Config{
		DexID: "stabull-test",
	}

	tracker, err := NewPoolTracker(config, client)
	require.NoError(t, err)

	// NZDS/USDC pool on Polygon
	poolAddress := "0xdcb7efACa996fe2985138bF31b647EFcd1D0901a"
	baseOracle := "0xa302a0b8a499fd0f00449df0a490dede21105955"  // NZDS/USD oracle
	quoteOracle := "0xfe4a8cc5b5b2366c1b58bea3858e81843581b2f7" // USDC/USD oracle

	// Execute
	ctx := context.Background()
	reserves, extra, err := tracker.fetchPoolStateWithOraclesFromNode(ctx, poolAddress, baseOracle, quoteOracle)

	// Assert
	require.NoError(t, err, "Should successfully fetch pool state with oracles")
	require.NotNil(t, reserves, "Reserves should not be nil")
	require.Len(t, reserves, 2, "Should have 2 reserves")

	t.Logf("=== Pool State with Oracle Rates ===")
	t.Logf("Pool: %s", poolAddress)

	t.Logf("\nReserves:")
	t.Logf("  Reserve 0: %s", reserves[0].String())
	t.Logf("  Reserve 1: %s", reserves[1].String())

	t.Logf("\nCurve Parameters:")
	t.Logf("  Alpha: %s", extra.CurveParams.Alpha)
	t.Logf("  Beta: %s", extra.CurveParams.Beta)
	t.Logf("  Delta: %s", extra.CurveParams.Delta)
	t.Logf("  Epsilon: %s", extra.CurveParams.Epsilon)
	t.Logf("  Lambda: %s", extra.CurveParams.Lambda)

	t.Logf("\nOracle Information:")
	t.Logf("  Base Oracle Address: %s", extra.BaseOracleAddress)
	t.Logf("  Quote Oracle Address: %s", extra.QuoteOracleAddress)

	if extra.BaseOracleRate != "" {
		t.Logf("  Base Oracle Rate (NZDS/USD): %s", extra.BaseOracleRate)
	}

	if extra.QuoteOracleRate != "" {
		t.Logf("  Quote Oracle Rate (USDC/USD): %s", extra.QuoteOracleRate)
	}

	if extra.OracleRate != "" {
		t.Logf("  Derived Oracle Rate (NZDS/USDC): %s", extra.OracleRate)
	}

	// Validate oracle data is present
	require.NotEmpty(t, extra.BaseOracleRate, "Base oracle rate should not be empty")
	require.NotEmpty(t, extra.QuoteOracleRate, "Quote oracle rate should not be empty")
	require.NotEmpty(t, extra.OracleRate, "Derived oracle rate should not be empty")

	// Pretty print the full Extra as JSON
	extraJSON, _ := json.MarshalIndent(extra, "", "  ")
	t.Logf("\nFull Extra JSON:\n%s", string(extraJSON))
}
