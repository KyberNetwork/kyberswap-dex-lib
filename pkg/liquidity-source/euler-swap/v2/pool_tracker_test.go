package v2

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/euler-swap/internal/rpcreplay"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/euler-swap/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// TestTrackVaultWithoutOracle tracks the Ethereum pool 0x0225eac3 at block 26340400 from a fixture of the node's
// answers. A vault without an oracle gets no price calls; the tracker used to grow each collateral's price row only up
// to the last vault with one, and indexed past its end (index out of range) when a later vault had none.
//
// Set EULER_RECORD_RPC to a node at that block to record missing calls into the fixture.
func TestTrackVaultWithoutOracle(t *testing.T) {
	srv := rpcreplay.New(t, "testdata/ethereum-26340400-0x0225eac3.json", os.Getenv("EULER_RECORD_RPC"))
	defer srv.Save()
	ctx := context.Background()
	client := srv.Client()
	cfg := shared.Config{DexID: DexType}
	pools, err := NewPoolsListUpdater(&cfg, client).initPools(ctx,
		[]common.Address{common.HexToAddress("0x0225eac3d800a8de0954dd046d13dd21546228a8")})
	require.NoError(t, err)
	tracker, err := NewPoolTracker(&cfg, client)
	require.NoError(t, err)
	tracked, err := tracker.GetNewPoolState(ctx, pools[0], pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	sim, err := NewPoolSimulator(tracked)
	require.NoError(t, err)

	// The pool's computeQuote at that block, for 0.001 of each token.
	const (
		weth  = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
		token = "0xe1b4d34e8754600962cd944b535180bd758e6c2e"
	)
	for _, tc := range []struct{ in, out, amount, want string }{
		{weth, token, "1000000000000000", "948745005077017"},
		{token, weth, "1000000000000000", "1053813200227429"},
	} {
		amount, _ := new(big.Int).SetString(tc.amount, 10)
		res, err := sim.CloneState().CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: tc.in, Amount: amount}, TokenOut: tc.out})
		if err != nil {
			t.Errorf("%s -> %s %s: %v", tc.in, tc.out, tc.amount, err)
			continue
		}
		require.Equal(t, tc.want, res.TokenAmountOut.Amount.String(), "%s -> %s %s", tc.in, tc.out, tc.amount)
	}
}
