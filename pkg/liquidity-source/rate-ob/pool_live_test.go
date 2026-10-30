package rateob

import (
	"context"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// TestLive lists every pair, tracks it and quotes both directions against a live chain. Set
// RATE_OB_RPC (and optionally RATE_OB_ENGINE / RATE_OB_EXECUTOR); defaults are Robinhood Chain
// Testnet (46630). Point it at an anvil fork: on the Arbitrum Orbit chain itself Multicall3 reports
// the parent chain's block.number, so the tracker's pinned read needs ArbMulticall2 there.
func TestLive(t *testing.T) {
	rpc := os.Getenv("RATE_OB_RPC")
	if rpc == "" || os.Getenv("CI") != "" {
		t.Skip("set RATE_OB_RPC to run")
	}
	cfg := &Config{DexID: DexType, ChainID: 46630,
		Engine:   envOr("RATE_OB_ENGINE", "0x611eec25268ac33061a88fc38c11466ad881e575"),
		Executor: envOr("RATE_OB_EXECUTOR", "0x000000000000000000000000000000000000dEaD")}
	client := ethrpc.New(rpc).SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	ctx := context.Background()

	pools, _, err := NewPoolListUpdater(cfg, client).GetNewPools(ctx, nil)
	require.NoError(t, err)
	require.NotEmpty(t, pools)
	tracker := NewPoolTracker(cfg, client)
	for _, p := range pools {
		p, err := tracker.GetNewPoolState(ctx, p, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
		t.Logf("%s static=%s block=%d reserves=%v extra=%s", p.Address, p.StaticExtra, p.BlockNumber, p.Reserves, p.Extra)
		sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
		require.NoError(t, err)
		for idxIn, amounts := range [][]string{{"1000000000000000000", "1000000000000000000000", "1000000000000000000000000"}, {"10000", "1000000", "100000000"}} {
			for _, amount := range amounts {
				res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: p.Tokens[idxIn].Address, Amount: bignumber.NewBig10(amount)},
					TokenOut:      p.Tokens[1-idxIn].Address})
				if err != nil {
					t.Logf("  in%d %s: %v", idxIn, amount, err)
					continue
				}
				t.Logf("  in%d %s: out=%s rem=%s gas=%d si=%+v", idxIn, amount, res.TokenAmountOut.Amount,
					res.RemainingTokenAmountIn.Amount, res.Gas, res.SwapInfo)
			}
		}
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
