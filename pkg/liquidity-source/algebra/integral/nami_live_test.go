package integral

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// The e2e pool plus launch pools (which must be detected as unsupported while the anti-sniper
// window is configured, per the Nami README).
var testNamiPools = []string{
	"0x6dbf1772aee6f453c6dafe705bb388685d5df1ea", // the pool pinned by TestNamiHistoricalTrackerQuote
}

var testNamiLaunchPools = []string{
	"0x8888cdad9ef66382f89e68f95ec384d89ec28888",
	"0x11110ecc00d927f70be6e35adb24bb7d3dc11111",
}

const (
	testNamiBasePluginFactory        = "0x8b7e4b0eb40ecc9db79b84d354fb54f48c7f8d3e"
	testNamiBasePluginFactoryLaunch  = "0xa9d6b74684c699081b893c5744d8a8f9a51f1650"
	testNamiFullPluginABIGist        = "/tmp/opencode/nami-abi/bf4943f122dd57ed76db0cf314904fc8-00c65e11b60f6cb7f14873d73d7cd9b700b7b237/INamiPluginV1Diamond.abi.json"
	testNamiFullPluginABIGistEnvName = "NAMI_PLUGIN_ABI_GIST"
)

// TestNamiLiveSwapFeeMatch is an opt-in differential test: it tracks pool state through the real
// tracker at a pinned block, mirrors the plugin's beforeSwap fee computation against that state,
// and requires it to equal on-chain swapFee(zeroToOne) evaluated in the same block. Launch pools
// are checked for the anti-sniper unsupported marking instead of a fee match.
func TestNamiLiveSwapFeeMatch(t *testing.T) {
	rpcURL := os.Getenv("NAMI_E2E_BASE_RPC")
	if rpcURL == "" {
		t.Skip("set NAMI_E2E_BASE_RPC to a Base archive RPC")
	}

	ctx := context.Background()
	rpc, err := ethclient.DialContext(ctx, rpcURL)
	require.NoError(t, err)
	t.Cleanup(rpc.Close)
	head, err := rpc.HeaderByNumber(ctx, nil)
	require.NoError(t, err)

	eth := ethrpc.NewWithClient(rpc).SetMulticallContract(common.HexToAddress("0xca11bde05977b3631167028862be2a173976ca11"))
	pinned := head.Number.Uint64() - 1 // the newest state a view call at pin can reach
	header, err := rpc.HeaderByNumber(ctx, big.NewInt(int64(pinned)))
	require.NoError(t, err)

	// discover recently created regular and launch Nami pools from PluginCreated
	discoveredRegular := discoverNamiPools(ctx, rpc, head, testNamiBasePluginFactory)
	discoveredLaunch := discoverNamiPools(ctx, rpc, head, testNamiBasePluginFactoryLaunch)
	t.Logf("discovered %d regular and %d launch pools from PluginCreated events",
		len(discoveredRegular), len(discoveredLaunch))

	for _, poolAddr := range dedupPools(append(testNamiPools, discoveredRegular...)) {
		poolAddr := poolAddr
		t.Run(poolAddr, func(t *testing.T) {
			runNamiFeeMatch(t, ctx, eth, header, poolAddr, nil)
		})
	}

	// fee override scenarios written into the deployed plugin's namespaced storage via state
	// overrides: each run compares the Go fee against on-chain swapFee for both directions
	t.Run("override-scenarios", func(t *testing.T) {
		var pluginAddr common.Address
		req := eth.NewRequest().SetContext(ctx).SetBlockNumber(header.Number)
		req.AddCall(&ethrpc.Call{ABI: poolV12ABI, Target: dedupPools(testNamiPools)[0], Method: poolPluginMethod}, []any{&pluginAddr})
		_, err := req.Aggregate()
		require.NoError(t, err)
		require.NotEqual(t, common.Address{}, pluginAddr)

		ts := uint64(header.Time)
		// floor, cap, premium0For1, premium1For0, decayWindow, fee0For1, fee1For0, expiresAt, maxDiscountBps
		scenarios := []struct {
			name string
			pack [9]uint64
		}{
			{"active override above side", [9]uint64{100, 5000, 300, 700, 3600, 6000, 3000, ts + 1000, 1000}},
			{"active override below lowest (discount clamp)", [9]uint64{100, 5000, 300, 700, 3600, 500, 300, ts + 1000, 1000}},
			{"decaying below side", [9]uint64{100, 5000, 300, 700, 3600, 500, 300, ts - 1000, 1000}},
			{"decaying above side", [9]uint64{100, 5000, 300, 700, 3600, 6000, 3000, ts - 1000, 1000}},
			{"floor clamps low override", [9]uint64{2000, 8000, 0, 0, 0, 0, 0, ts + 1000, 0}},
			{"expired override past decay window", [9]uint64{100, 5000, 300, 700, 3600, 6000, 3000, ts - 7200, 1000}},
		}
		for _, scenario := range scenarios {
			scenario := scenario
			t.Run(scenario.name, func(t *testing.T) {
				overrides := map[common.Address]gethclient.OverrideAccount{
					pluginAddr: {StateDiff: map[common.Hash]common.Hash{
						feeOverrideNamespaceSlot: packNamiFeeLayout(scenario.pack)},
					},
				}
				runNamiFeeMatch(t, ctx, eth, header, dedupPools(testNamiPools)[0], overrides)
			})
		}
	})

	// the anti-sniper detection must flag every live launch pool. The launch pools are only
	// created at the launch itself (per the Nami team), so a missing pool is a skip, not a failure
	launchPools := dedupPools(append(testNamiLaunchPools, discoveredLaunch...))
	require.NotEmpty(t, launchPools, "no launch pools to check")
	for _, poolAddr := range launchPools {
		poolAddr := poolAddr
		t.Run(poolAddr+"/launch-detection", func(t *testing.T) {
			pinNamiClock(t, uint32(header.Time))
			tracker := &PoolTracker{EthrpcClient: eth, config: &Config{DexID: "nami-v4"}}
			p := entity.Pool{Address: poolAddr, Tokens: []*entity.PoolToken{
				{Address: "0x8888cdad9ef66382f89e68f95ec384d89ec28888"}, {Address: "0x4200000000000000000000000000000000000006"}}}
			state, err := tracker.FetchRPCData(ctx, &p, pinned, nil)
			if err != nil {
				t.Skipf("launch pool does not exist on-chain yet: %v", err)
			}
			require.NotNil(t, state.NamiFee)
			require.NotEmpty(t, state.NamiFee.Unsupported,
				"launch pool must be marked unsupported while the anti-sniper window is configured")
			t.Logf("launch pool marked unsupported: %s", state.NamiFee.Unsupported)
		})
	}
}

func runNamiFeeMatch(t *testing.T, ctx context.Context, eth *ethrpc.Client, header *types.Header, poolAddr string,
	overrides map[common.Address]gethclient.OverrideAccount) {
	block := header.Number
	pinNamiClock(t, uint32(header.Time))

	// read the plugin address of the pool in the same pinned block
	var pluginAddr common.Address
	req := eth.NewRequest().SetContext(ctx).SetBlockNumber(block).SetOverrides(overrides)
	req.AddCall(&ethrpc.Call{ABI: poolV12ABI, Target: poolAddr, Method: poolPluginMethod}, []any{&pluginAddr})
	_, err := req.Aggregate()
	require.NoError(t, err)
	if pluginAddr == (common.Address{}) {
		t.Skip("pool has no plugin")
	}

	tracker := &PoolTracker{EthrpcClient: eth, config: &Config{DexID: "nami-v4"}}
	p := entity.Pool{Address: poolAddr,
		Tokens: []*entity.PoolToken{{Address: "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"},
			{Address: "0xb2000000000000000000008bc8786b856e61707c"}}}
	state, err := tracker.FetchRPCData(ctx, &p, block.Uint64(), overrides)
	require.NoError(t, err)
	require.NotNil(t, state.NamiFee, "nami fee config missing")
	if state.NamiFee.Unsupported != "" {
		t.Skipf("pool unsupported by the integration: %s", state.NamiFee.Unsupported)
	}

	sim := ps.CloneState().(*PoolSimulator)
	sim.globalState = state.State
	sim.dynamicFee = state.DynamicFee
	sim.volatilityOracle = state.VolatilityOracle
	sim.timepoints = NewTimepointStorage(state.Timepoints)
	sim.namiFee = state.NamiFee

	fee01, _, err := sim.beforeNamiSwap(true)
	require.NoError(t, err)
	fee10, _, err := sim.beforeNamiSwap(false)
	require.NoError(t, err)

	// the same block's on-chain swapFee for both directions, queried with the full
	// (deployed) interface ABI from the gist rather than the minimal fetch ABI
	pluginABIPath := os.Getenv(testNamiFullPluginABIGistEnvName)
	if pluginABIPath == "" {
		pluginABIPath = testNamiFullPluginABIGist
	}
	pluginABIBytes, err := os.ReadFile(pluginABIPath)
	if err != nil {
		t.Skipf("full Nami plugin ABI (gist) not found: %v", err)
	}
	fullABI, err := abi.JSON(bytes.NewReader(pluginABIBytes))
	require.NoError(t, err)

	var want01, want10 uint16
	plugin := pluginAddr.Hex()
	req = eth.NewRequest().SetContext(ctx).SetBlockNumber(block).SetOverrides(overrides)
	req.AddCall(&ethrpc.Call{ABI: fullABI, Target: plugin, Method: "swapFee", Params: []any{true}}, []any{&want01})
	req.AddCall(&ethrpc.Call{ABI: fullABI, Target: plugin, Method: "swapFee", Params: []any{false}}, []any{&want10})
	_, err = req.Aggregate()
	require.NoError(t, err)

	require.EqualValues(t, want01, fee01, "zeroToOne fee mismatch")
	require.EqualValues(t, want10, fee10, "oneForZero fee mismatch")
	t.Logf("matched: fee(0->1)=%d fee(1->0)=%d", fee01, fee10)
	fmt.Printf("pool %s block %d: fee(0->1)=%d fee(1->0)=%d matched on-chain\n", poolAddr, block, fee01, fee10)
}

// feeOverrideNamespaceSlot is libraries.FeeOverrideStorage.NAMESPACE, slot 0 packs the bounds,
// premiums, live override, source flag and discount allowance.
var feeOverrideNamespaceSlot = common.HexToHash("0x4e6a83909802af262770f859c245d2a01fddbd871c0bec861e01a093143bd600")

// packNamiFeeLayout packs the FeeOverrideStorage.Layout slot 0 fields (floor, cap, premiums,
// decayWindow, fee0For1, fee1For0, expiresAt, maxDiscountBps) in Solidity storage order.
func packNamiFeeLayout(v [9]uint64) common.Hash {
	word := new(big.Int).SetUint64(v[0]) // floor
	for i, shift := range []uint{16, 32, 48, 64, 96, 112} {
		word = word.Or(word, new(big.Int).Lsh(new(big.Int).SetUint64(v[i+1]), shift))
	}
	word = word.Or(word, new(big.Int).Lsh(new(big.Int).SetUint64(v[7]), 128)) // expiresAt (uint40)
	word = word.Or(word, new(big.Int).Lsh(new(big.Int).SetUint64(v[8]), 176)) // maxDiscountBps
	return common.BigToHash(word)
}

// pinNamiClock pins the package-level simulator clock to the pinned block's timestamp, so both
// sides evaluate at the exact time the on-chain view call sees.
func pinNamiClock(t *testing.T, ts uint32) {
	t.Helper()
	oldTime := blockTimestamp
	blockTimestamp = func() uint32 { return ts }
	t.Cleanup(func() { blockTimestamp = oldTime })
}

// discoverNamiPools scans PluginCreated logs of a plugin factory in chunks, tolerating RPC range
// caps by keeping what has been found so far.
func discoverNamiPools(ctx context.Context, rpc *ethclient.Client, head *types.Header, factory string) []string {
	var pools []string
	const chunk = uint64(50_000)
	const span = uint64(500_000)
	for end := head.Number.Uint64(); end+chunk > head.Number.Uint64()-span; end -= chunk {
		start := end - chunk + 1
		if start < 1 {
			break
		}
		logs, err := rpc.FilterLogs(ctx, ethereum.FilterQuery{
			FromBlock: big.NewInt(int64(start)), ToBlock: big.NewInt(int64(end)),
			Addresses: []common.Address{common.HexToAddress(factory)},
		})
		if err != nil {
			break
		}
		for _, lg := range logs {
			if len(lg.Topics) >= 2 {
				pools = append(pools, common.BytesToAddress(lg.Topics[1][:]).Hex())
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return pools
}

func dedupPools(pools []string) []string {
	seen := make(map[string]bool, len(pools))
	out := make([]string, 0, len(pools))
	for _, p := range pools {
		if p := common.HexToAddress(p).Hex(); !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return out
}
