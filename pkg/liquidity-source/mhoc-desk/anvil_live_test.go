package mhocdesk

import (
	"math/big"
	"math/rand/v2"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// liveCompare finds the pool through the list updater (from the registry when one is given, else from the static
// list), tracks it, builds the simulator from what was read, and compares it with eth_call quote() at the tracked
// block for n log-uniform amounts in both directions (an unfillable amount is 0 on chain and an error in the
// simulator).
func liveCompare(t *testing.T, client *ethrpc.Client, poolAddr, registry string, n int) (*PoolSimulator, entity.Pool,
	int) {
	cfg := &Config{DexID: DexType, Pools: []string{poolAddr}}
	if registry != "" {
		cfg = &Config{DexID: DexType, Registry: registry}
	}
	pools, _, err := NewPoolsListUpdater(cfg, client).GetNewPools(t.Context(), nil)
	require.NoError(t, err)
	i := slices.IndexFunc(pools, func(p entity.Pool) bool { return strings.EqualFold(p.Address, poolAddr) })
	require.GreaterOrEqual(t, i, 0, "pool %s not found among %d pools", poolAddr, len(pools))
	tr, _ := NewPoolTracker(cfg, client)
	tracked, err := tr.GetNewPoolState(t.Context(), pools[i], pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	sim, err := NewPoolSimulator(tracked)
	require.NoError(t, err)
	bn, ts := tracked.BlockNumber, uint64(tracked.Timestamp)
	return sim, tracked, compareAt(t, client.GetETHClient(), sim, tracked, poolAddr, n, bn, ts, sim.calcExact)
}

func compareAt(t *testing.T, eth *ethclient.Client, sim *PoolSimulator, tracked entity.Pool, poolAddr string, n int,
	bn, ts uint64, calc func(pool.CalcAmountOutParams, uint64, uint64) (*pool.CalcAmountOutResult, error)) int {
	target := common.HexToAddress(poolAddr)
	block := new(big.Int).SetUint64(bn)
	onchain := func(tokenIn string, amt *big.Int) *big.Int {
		data, err := propAMMABI.Pack("quote", common.HexToAddress(tokenIn), amt)
		require.NoError(t, err)
		ret, err := eth.CallContract(t.Context(), ethereum.CallMsg{To: &target, Data: data}, block)
		require.NoError(t, err)
		out, err := propAMMABI.Unpack("quote", ret)
		require.NoError(t, err)
		return out[0].(*big.Int)
	}
	rng := rand.New(rand.NewPCG(bn, ts))
	tokens := sim.GetTokens()
	filled := 0
	for i := 0; i < n; i++ {
		in := i % 2
		// log-uniform raw amounts from 1 to ~10^(decimals+9): every branch from NoFill to ExceedsMaxSize
		dec := int(tracked.Tokens[in].Decimals)
		amt := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(rng.IntN(dec+10))), nil)
		amt.Mul(amt, big.NewInt(int64(1+rng.IntN(999))))
		want := onchain(tokens[in], amt)
		res, err := calc(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: tokens[in], Amount: amt}, TokenOut: tokens[1-in]}, bn, ts)
		got := big.NewInt(0)
		if err == nil {
			got = res.TokenAmountOut.Amount
			filled++
		}
		require.Equal(t, want.String(), got.String(), "block %d tokenIn %d amount %s (sim err %v)", bn, in, amt, err)
	}
	return filled
}

// TestAnvilLive runs the list updater and the tracker against a real PropAMM on a local anvil chain, builds the
// simulator from what they read, and compares it with eth_call quote() for a few hundred amounts in both
// directions: at the tracked block, then (after mining one empty block) in the next block through CalcAmountOut's
// own clock, which is what routing prices on. The pool is found through the registry when MHOCDESK_ANVIL_REGISTRY is
// set. Skipped unless driven by the desk's anvil_live.py, which sets the variables:
//
//	python3 anvil_live.py --rounds 10 -- go test ./pkg/liquidity-source/mhoc-desk/ -run TestAnvilLive -count=1
func TestAnvilLive(t *testing.T) {
	url, poolAddr, mc := os.Getenv("MHOCDESK_ANVIL_RPC"), os.Getenv("MHOCDESK_ANVIL_POOL"), os.Getenv("MHOCDESK_ANVIL_MULTICALL")
	if url == "" || poolAddr == "" || mc == "" {
		t.Skip("set by anvil_live.py")
	}
	client := ethrpc.New(url).SetMulticallContract(common.HexToAddress(mc))
	sim, tracked, filled := liveCompare(t, client, poolAddr, os.Getenv("MHOCDESK_ANVIL_REGISTRY"), 300)
	t.Logf("block %d: 300 quotes match eth_call quote() (%d fillable)", tracked.BlockNumber, filled)

	// one empty block later (anvil only): the state is the same, the per-block counter has restarted
	rpc := client.GetETHClient().Client()
	ts1 := uint64(tracked.Timestamp) + 1
	require.NoError(t, rpc.CallContext(t.Context(), nil, "evm_setNextBlockTimestamp", hexutil.Uint64(ts1)))
	require.NoError(t, rpc.CallContext(t.Context(), nil, "evm_mine"))
	sim.quoteAt = int64(ts1)
	filled = compareAt(t, client.GetETHClient(), sim, tracked, poolAddr, 300, tracked.BlockNumber+1, ts1,
		func(p pool.CalcAmountOutParams, _, _ uint64) (*pool.CalcAmountOutResult, error) {
			return sim.CalcAmountOut(p)
		})
	t.Logf("block %d: 300 next-block quotes (CalcAmountOut) match eth_call quote() (%d fillable)",
		tracked.BlockNumber+1, filled)
}

// TestBaseMainnetLive reads a deployed pool on Base (8453) read-only through MHOCDESK_BASE_RPC (for example
// https://mainnet.base.org) and Multicall3, and checks the simulator against eth_call quote() at the tracked block.
// The pool is MHOCDESK_BASE_POOL, found through the registry MHOCDESK_BASE_REGISTRY when that is set. Before the
// owner configures a pool (paused, every cap 0, no inventory) quote() is 0 for every amount and the simulator must
// report no liquidity; afterwards the same comparison checks real fills. Skipped unless the variables are set.
func TestBaseMainnetLive(t *testing.T) {
	url, poolAddr := os.Getenv("MHOCDESK_BASE_RPC"), os.Getenv("MHOCDESK_BASE_POOL")
	if url == "" || poolAddr == "" {
		t.Skip("set MHOCDESK_BASE_RPC and MHOCDESK_BASE_POOL (and optionally MHOCDESK_BASE_REGISTRY)")
	}
	client := ethrpc.New(url).SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	_, tracked, filled := liveCompare(t, client, poolAddr, os.Getenv("MHOCDESK_BASE_REGISTRY"), 40)
	t.Logf("Base block %d, pool %s (%s/%s): 40 quotes match eth_call quote() (%d fillable; 0 = no liquidity yet)",
		tracked.BlockNumber, tracked.Address, tracked.Tokens[0].Address, tracked.Tokens[1].Address, filled)
}
