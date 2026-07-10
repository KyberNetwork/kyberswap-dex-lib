package umbraedlmm

import (
	"context"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Canonical Base mainnet U1/WETH DLMM pair (binStep 25), its factory and the PairViewer.
const (
	verifyPair     = "0x697b72320656e6dc60db7a4bfb95084c9d9c55a0"
	verifyFactory  = "0x17Da44dcbdffD8c715be7A368E19F252C2940Fee"
	verifyViewer   = "0xbA3A5400A95b055b1412e18ad1978EdF32Fc3F05"
	verifyRouter   = "0x4965DD6877ca9DE77caca2f57996651e7AF23c93"
	multicall3Base = "0xcA11bde05977b3631167028862bE2a173976CA11"
)

// TestVerifyAgainstChain is the end-to-end gate: real PoolTracker -> PoolSimulator -> compared to
// the pair's on-chain quoteSwap (via PairViewer) across a spread of amounts in both directions, all
// pinned to the tracked block. Set UMBRAE_BASE_RPC_URL to run.
func TestVerifyAgainstChain(t *testing.T) {
	rpcURL := os.Getenv("UMBRAE_BASE_RPC_URL")
	if rpcURL == "" {
		t.Skip("UMBRAE_BASE_RPC_URL not set; skipping live verification")
	}
	client := ethrpc.New(rpcURL).SetMulticallContract(common.HexToAddress(multicall3Base))
	ctx := context.Background()

	// Full ingestion pipeline: static config (as the lister persists it) -> tracker -> simulator.
	sim, tokenX, tokenY, pinBlock := buildLiveSimulator(t, ctx, client)
	require.Equal(t, verifyRouter, sim.GetApprovalAddress(tokenX, tokenY),
		"approval address must be the DLMM Router, not the pair")

	amounts := []*big.Int{exp10(14), exp10(15), exp10(16), exp10(17), exp10(18), mul(exp10(18), 10), mul(exp10(18), 1000)}
	for _, swapForY := range []bool{true, false} {
		tokenIn, tokenOut := tokenX, tokenY
		if !swapForY {
			tokenIn, tokenOut = tokenY, tokenX
		}
		dir := "X->Y"
		if !swapForY {
			dir = "Y->X"
		}
		for _, amt := range amounts {
			chainOut, consumed, binsTrav, _ := viewerQuoteSwap(t, ctx, client, pinBlock, swapForY, amt)
			res, cerr := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amt}, TokenOut: tokenOut,
			})

			if consumed.Cmp(amt) < 0 {
				t.Logf("%s in=%-24s | consumed=%-24s (partial) bins=%-3s -> sim err=%v", dir, amt, consumed, binsTrav, cerr)
				require.ErrorIs(t, cerr, ErrInsufficientLiquidity,
					"%s in=%s: viewer partial-filled (swap() reverts), sim must reject", dir, amt)
				continue
			}
			require.NoError(t, cerr, "%s in=%s: fully-consumable swap must succeed", dir, amt)
			require.Equal(t, chainOut.String(), res.TokenAmountOut.Amount.String(),
				"%s in=%s: sim output must match on-chain quoteSwap", dir, amt)
			t.Logf("%s in=%-24s | chain=%-24s sim=%-24s bins=%-3s OK", dir, amt, chainOut, res.TokenAmountOut.Amount, binsTrav)
		}
	}
}

// TestVerifyListerLive exercises the real PoolsListUpdater against the live DLMM factory: enumerate
// pairs, then assert the canonical U1/WETH pair is discovered with correctly-persisted StaticExtra
// (binStep, decimals, router). This is the on-chain discovery path the verification test skips (it
// builds the entity.Pool inline). Set UMBRAE_BASE_RPC_URL to run.
func TestVerifyListerLive(t *testing.T) {
	rpcURL := os.Getenv("UMBRAE_BASE_RPC_URL")
	if rpcURL == "" {
		t.Skip("UMBRAE_BASE_RPC_URL not set; skipping live lister verification")
	}
	client := ethrpc.New(rpcURL).SetMulticallContract(common.HexToAddress(multicall3Base))
	ctx := context.Background()

	lister := NewPoolsListUpdater(&Config{
		DexID: DexType, FactoryAddress: verifyFactory, ViewerAddress: verifyViewer,
		RouterAddress: verifyRouter,
	}, client)

	pools, _, err := lister.GetNewPools(ctx, nil)
	require.NoError(t, err)
	require.NotEmpty(t, pools, "factory enumeration returned no pairs")
	t.Logf("lister discovered %d pair(s)", len(pools))

	var found *entity.Pool
	for i := range pools {
		if strings.EqualFold(pools[i].Address, verifyPair) {
			found = &pools[i]
			break
		}
	}
	require.NotNil(t, found, "canonical U1/WETH pair not enumerated by the factory lister")

	var se StaticExtra
	require.NoError(t, json.Unmarshal([]byte(found.StaticExtra), &se))
	require.Equal(t, uint16(25), se.BinStep, "binStep")
	require.Equal(t, verifyRouter, se.Router, "router persisted into StaticExtra")
	require.Len(t, found.Tokens, 2)
	require.NotEmpty(t, found.Tokens[0].Address)
	require.NotEmpty(t, found.Tokens[1].Address)
	require.Equal(t, entity.PoolReserves{"0", "0"}, found.Reserves, "lister leaves reserves at 0 for the tracker")
	t.Logf("canonical pair OK: binStep=%d decX=%d decY=%d router=%s tokens=[%s,%s]",
		se.BinStep, se.DecimalsX, se.DecimalsY, se.Router, found.Tokens[0].Address, found.Tokens[1].Address)
}

func viewerQuoteSwap(t *testing.T, ctx context.Context, client *ethrpc.Client, block *big.Int, swapForY bool, amountIn *big.Int) (amountOut, consumedAmountIn, binsTrav, finalBinID *big.Int) {
	t.Helper()
	var out struct {
		AmountOut        *big.Int
		ConsumedAmountIn *big.Int
		FinalBinId       *big.Int
		BinsTraversed    *big.Int
		TotalFee         *big.Int
	}
	_, err := client.R().SetContext(ctx).SetBlockNumber(block).AddCall(&ethrpc.Call{
		ABI: viewerABI, Target: verifyViewer, Method: viewerMethodQuoteSwap,
		Params: []any{common.HexToAddress(verifyPair), swapForY, amountIn},
	}, []any{&out}).Call()
	require.NoError(t, err)
	return out.AmountOut, out.ConsumedAmountIn, out.BinsTraversed, out.FinalBinId
}

// TestVerifyUpdateBalance proves the post-swap state transition KyberSwap relies on for multi-hop /
// split routing: (1) CalcAmountOut's SwapInfo.newActiveID matches the pair's on-chain finalBinId,
// (2) after UpdateBalance the bins the swap fully crossed are emptied on the output side, and (3) a
// second CalcAmountOut reads the advanced book (starts at the new active bin, never re-quotes the
// already-consumed liquidity). Set UMBRAE_BASE_RPC_URL to run.
func TestVerifyUpdateBalance(t *testing.T) {
	rpcURL := os.Getenv("UMBRAE_BASE_RPC_URL")
	if rpcURL == "" {
		t.Skip("UMBRAE_BASE_RPC_URL not set; skipping live update-balance verification")
	}
	client := ethrpc.New(rpcURL).SetMulticallContract(common.HexToAddress(multicall3Base))
	ctx := context.Background()
	sim, tokenX, tokenY, pinBlock := buildLiveSimulator(t, ctx, client)

	// Y->X is the liquid direction for this pool; pick amounts that cross several bins.
	for _, amt := range []*big.Int{exp10(15), exp10(16), exp10(17)} {
		_, _, _, chainFinalBin := viewerQuoteSwap(t, ctx, client, pinBlock, false, amt)

		hop := sim.CloneState() // never mutate the shared base
		res, err := hop.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: tokenY, Amount: amt}, TokenOut: tokenX,
		})
		require.NoError(t, err)
		si := res.SwapInfo.(SwapInfo)
		require.Equal(t, chainFinalBin.Uint64(), uint64(si.newActiveID),
			"post-swap active bin must match the pair's finalBinId (amt=%s)", amt)

		// Apply the trade, then assert UpdateBalance wrote back CalcAmountOut's own per-bin deltas
		// verbatim (consumes SwapInfo, never recomputes) and advanced the active bin.
		hop.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: si})
		mutated := hop.(*PoolSimulator)
		require.Equal(t, si.newActiveID, mutated.activeID, "UpdateBalance must advance activeID")
		for _, u := range si.binUpdates {
			require.Equal(t, u.reserveX, mutated.bins[u.index].ReserveX, "bin %d reserveX after update", mutated.bins[u.index].ID)
			require.Equal(t, u.reserveY, mutated.bins[u.index].ReserveY, "bin %d reserveY after update", mutated.bins[u.index].ID)
		}

		// A second swap must start from the advanced book: the same input now yields no more than the
		// first hop did (consumed liquidity is gone) and still lands at-or-beyond the new active bin.
		res2, err := hop.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: tokenY, Amount: amt}, TokenOut: tokenX,
		})
		if err == nil {
			require.LessOrEqual(t, res2.TokenAmountOut.Amount.Cmp(res.TokenAmountOut.Amount), 0,
				"second hop must not out-quote the first on a depleted book (amt=%s)", amt)
			si2 := res2.SwapInfo.(SwapInfo)
			require.GreaterOrEqual(t, uint64(si2.newActiveID), uint64(si.newActiveID),
				"second hop must continue from the advanced active bin (amt=%s)", amt)
		}
		t.Logf("Y->X amt=%-22s finalBin chain=%d sim=%d out1=%s", amt, chainFinalBin, si.newActiveID, res.TokenAmountOut.Amount)
	}
}

// buildLiveSimulator runs the real lister-equivalent static config + tracker -> simulator pipeline
// against the canonical pair, returning the simulator, its token addresses and the pinned block.
func buildLiveSimulator(t *testing.T, ctx context.Context, client *ethrpc.Client) (*PoolSimulator, string, string, *big.Int) {
	t.Helper()
	var (
		tokenXAddr, tokenYAddr common.Address
		dec                    decimalsResult
		binStep                uint16
	)
	_, err := client.R().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: pairABI, Target: verifyPair, Method: pairMethodTokenX}, []any{&tokenXAddr}).
		AddCall(&ethrpc.Call{ABI: pairABI, Target: verifyPair, Method: pairMethodTokenY}, []any{&tokenYAddr}).
		AddCall(&ethrpc.Call{ABI: pairABI, Target: verifyPair, Method: pairMethodBinStep}, []any{&binStep}).
		AddCall(&ethrpc.Call{ABI: pairABI, Target: verifyPair, Method: pairMethodGetDecimals}, []any{&dec}).
		Aggregate()
	require.NoError(t, err)

	se, _ := json.Marshal(StaticExtra{BinStep: binStep, DecimalsX: dec.DecimalsX, DecimalsY: dec.DecimalsY, Router: verifyRouter})
	ep := entity.Pool{
		Address: verifyPair, Exchange: DexType, Type: DexType,
		Reserves:    entity.PoolReserves{"0", "0"},
		Tokens:      []*entity.PoolToken{{Address: tokenXAddr.Hex()}, {Address: tokenYAddr.Hex()}},
		StaticExtra: string(se), Extra: "{}",
	}
	tracker := NewPoolTracker(&Config{DexID: DexType, FactoryAddress: verifyFactory, ViewerAddress: verifyViewer, RouterAddress: verifyRouter}, client)
	ep, err = tracker.GetNewPoolState(ctx, ep, pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	sim, err := NewPoolSimulator(ep)
	require.NoError(t, err)
	return sim, tokenXAddr.Hex(), tokenYAddr.Hex(), new(big.Int).SetUint64(ep.BlockNumber)
}

func exp10(n uint) *big.Int            { return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n)), nil) }
func mul(a *big.Int, n int64) *big.Int { return new(big.Int).Mul(a, big.NewInt(n)) }
