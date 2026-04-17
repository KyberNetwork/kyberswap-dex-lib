package fermi

import (
	"context"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
)

const (
	testTitanURLEU = "https://eu.rpc.titanbuilder.xyz"
	testTitanURLUS = "https://us.rpc.titanbuilder.xyz"
	testTitanURLAP = "https://ap.rpc.titanbuilder.xyz"
)

// newTestTracker returns a PoolTracker configured for live Titan HTTP RPC
// without an ethrpc client (use only for fetchStateOverrides / extractMidPrice).
func newTestTracker() *PoolTracker {
	cfg := &Config{
		FermiSwapper: fermiSwapperAddr,
		FermiEngine:  fermiEngineAddr,
		TraderVault:  fermiTraderVaultAddr,
		Titan: TitanConfig{
			URLs: []string{testTitanURLEU, testTitanURLAP, testTitanURLUS},
		},
	}
	return &PoolTracker{
		config:       cfg,
		titanClients: newTitanClients(cfg.Titan),
	}
}

// fetchTestOverrides calls Titan HTTP RPC and returns overrides + midPrice
// for the WETH/USDC pair. Fails the test if the RPC returns nothing.
func fetchTestOverrides(t *testing.T) (map[common.Address]gethclient.OverrideAccount, *big.Int) {
	t.Helper()
	tracker := newTestTracker()

	overrides := tracker.fetchStateOverrides(context.Background())
	require.NotNil(t, overrides, "titan HTTP RPC returned no overrides")

	weth := common.HexToAddress("0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2")
	usdc := common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")
	midPrice := tracker.extractMidPrice(weth, usdc, overrides)

	return overrides, midPrice
}

// fetchTestOverridesForPair calls Titan HTTP RPC and extracts midPrice for
// an arbitrary token pair.
func fetchTestOverridesForPair(
	t *testing.T,
	token0, token1 common.Address,
) (map[common.Address]gethclient.OverrideAccount, *big.Int) {
	t.Helper()
	tracker := newTestTracker()

	overrides := tracker.fetchStateOverrides(context.Background())
	require.NotNil(t, overrides, "titan HTTP RPC returned no overrides")

	midPrice := tracker.extractMidPrice(token0, token1, overrides)
	return overrides, midPrice
}

func TestToStateOverrides_PreservesAllSlots(t *testing.T) {
	weth := common.HexToAddress("0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2")
	usdc := common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")
	usdt := common.HexToAddress("0xdac17f958d2ee523a2206206994597c13d831ec7")
	engine := common.HexToAddress(fermiEngineAddr)

	pickPairBase := func(t0, t1 common.Address) common.Hash {
		fwd, _ := pairKeyForTokens(t0, t1)
		return pairBaseSlot(fwd)
	}
	baseUSDC := pickPairBase(weth, usdc)
	baseUSDT := pickPairBase(weth, usdt)
	require.NotEqual(t, baseUSDC, baseUSDT)

	field0Val := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001")
	priceUSDC := common.HexToHash("0x000000000000000000000000000000000000000000000000000000003131b764e0")
	priceUSDT := common.HexToHash("0x00000000000000000000000000000000000000000000000000000000beef0001")

	overrides := map[common.Address]gethclient.OverrideAccount{
		engine: {
			StateDiff: map[common.Hash]common.Hash{
				baseUSDC:                field0Val,
				slotOffset(baseUSDC, 1): priceUSDC,
				baseUSDT:                field0Val,
				slotOffset(baseUSDT, 1): priceUSDT,
			},
		},
	}

	tracker := &PoolTracker{config: &Config{FermiEngine: fermiEngineAddr}}

	priceA, baseA, okA := tracker.findPairOverride(weth, usdc, overrides)
	require.True(t, okA)
	require.Equal(t, baseUSDC, baseA)
	require.Equal(t, priceUSDC.Big(), priceA)

	priceB, baseB, okB := tracker.findPairOverride(weth, usdt, overrides)
	require.True(t, okB)
	require.Equal(t, baseUSDT, baseB)
	require.Equal(t, priceUSDT.Big(), priceB)

	so := toStateOverrides(overrides)
	require.Len(t, so, 1, "exactly one contract entry (FermiEngine)")
	var diff map[string]string
	for k, v := range so {
		require.True(t, strings.EqualFold(fermiEngineAddr, k))
		diff = v
	}
	require.Len(t, diff, 4, "all four slots must be present")
	require.Contains(t, diff, baseUSDC.Hex())
	require.Contains(t, diff, slotOffset(baseUSDC, 1).Hex())
	require.Contains(t, diff, baseUSDT.Hex())
	require.Contains(t, diff, slotOffset(baseUSDT, 1).Hex())
}

func TestToStateOverrides_EmptyInput(t *testing.T) {
	require.Nil(t, toStateOverrides(nil))
	require.Nil(t, toStateOverrides(map[common.Address]gethclient.OverrideAccount{}))
	require.Nil(t, toStateOverrides(map[common.Address]gethclient.OverrideAccount{
		common.HexToAddress(fermiEngineAddr): {StateDiff: nil},
	}))
}

func TestExtractMidPrice_NilOverrides(t *testing.T) {
	tracker := newTestTracker()
	weth := common.HexToAddress("0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2")
	usdc := common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")

	require.Nil(t, tracker.extractMidPrice(weth, usdc, nil))
	require.Nil(t, tracker.extractMidPrice(weth, usdc, map[common.Address]gethclient.OverrideAccount{}))
}

func TestLive_QuoteWithVsWithoutOverride(t *testing.T) {
	test.SkipCI(t)

	if testing.Short() {
		t.Skip("live network test")
	}
	rpcURL := envRPCURL()

	weth := common.HexToAddress("0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2")
	usdc := common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")

	overrides, midPrice := fetchTestOverrides(t)
	t.Logf("Titan overrides: engine slots patched, midPrice=%v", midPrice)

	ec, err := ethclient.Dial(rpcURL)
	require.NoError(t, err)
	defer ec.Close()

	rpcClient := ethrpc.NewWithClient(ec)
	doCall := func(useOverride bool) *big.Int {
		var res struct {
			AmountIn  *big.Int
			AmountOut *big.Int
		}
		req := rpcClient.NewRequest().SetContext(context.Background())
		req.AddCall(&ethrpc.Call{
			ABI:    fermiSwapperABI,
			Target: fermiSwapperAddr,
			Method: methodQuote,
			Params: []any{weth, usdc, big.NewInt(1_000_000_000_000_000_000)},
		}, []any{&res})
		if useOverride {
			req.SetOverrides(overrides)
		}
		_, err := req.Call()
		require.NoError(t, err, "eth_call (override=%t)", useOverride)
		return res.AmountOut
	}

	noOverride := doCall(false)
	withOverride := doCall(true)
	t.Logf("WITHOUT override: %s USDC (raw)", noOverride)
	t.Logf("WITH    override: %s USDC (raw)", withOverride)

	if noOverride.Cmp(withOverride) == 0 {
		t.Logf("identical — storage may have just been refreshed")
		return
	}
	t.Logf("drift = %s bps", bpsDrift(noOverride, withOverride))
}

func TestLive_OverridesHaveEngineSlots(t *testing.T) {
	if testing.Short() {
		t.Skip("live network test")
	}
	overrides, midPrice := fetchTestOverrides(t)
	engine := common.HexToAddress(fermiEngineAddr)

	acct, ok := overrides[engine]
	require.True(t, ok, "overrides must contain FermiEngine")
	require.NotEmpty(t, acct.StateDiff)
	t.Logf("engine slots patched: %d", len(acct.StateDiff))

	require.NotNil(t, midPrice)
	require.True(t, midPrice.Sign() > 0)
	t.Logf("midPrice = %s", midPrice)
}

func TestLive_SlotKeysMatchOverrides(t *testing.T) {
	if testing.Short() {
		t.Skip("live network test")
	}
	rpcURL := envRPCURL()

	weth := common.HexToAddress("0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2")
	usdc := common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")
	engine := common.HexToAddress(fermiEngineAddr)
	fwd, rev := pairKeyForTokens(weth, usdc)

	overrides, _ := fetchTestOverrides(t)
	acct := overrides[engine]

	var matchedKey common.Hash
	for _, pk := range []common.Hash{fwd, rev} {
		base := pairBaseSlot(pk)
		if _, found := acct.StateDiff[slotOffset(base, 1)]; found {
			matchedKey = pk
			break
		}
	}
	require.NotEqual(t, common.Hash{}, matchedKey, "overrides must contain WETH/USDC slots")

	base := pairBaseSlot(matchedKey)
	ec, err := ethclient.Dial(rpcURL)
	require.NoError(t, err)
	defer ec.Close()

	ctx := t.Context()
	t5Raw, err := ec.StorageAt(ctx, engine, slotOffset(base, 5), nil)
	require.NoError(t, err)
	t6Raw, err := ec.StorageAt(ctx, engine, slotOffset(base, 6), nil)
	require.NoError(t, err)

	tokenIn := common.BytesToAddress(t5Raw)
	tokenOut := common.BytesToAddress(t6Raw)
	t.Logf("field5=%s  field6=%s", tokenIn.Hex(), tokenOut.Hex())
	require.True(t, (tokenIn == weth && tokenOut == usdc) || (tokenIn == usdc && tokenOut == weth))
}

func TestLive_MidPriceFromOverrides(t *testing.T) {
	if testing.Short() {
		t.Skip("live network test")
	}
	_, midPrice := fetchTestOverrides(t)
	require.NotNil(t, midPrice)

	lower := big.NewInt(10_000_000_000)     // $100 in 1e8
	upper := big.NewInt(10_000_000_000_000) // $100 000 in 1e8
	require.True(t, midPrice.Cmp(lower) >= 0 && midPrice.Cmp(upper) <= 0,
		"midPrice %s out of expected range", midPrice)
	t.Logf("midPrice = %s (≈ $%.2f)", midPrice, float64(midPrice.Int64())/1e8)
}

// TestLive_USDT_QuoteWithOverride runs the WETH/USDT pair through the same
// override vs plain comparison.
func TestLive_USDT_QuoteWithOverride(t *testing.T) {
	if testing.Short() {
		t.Skip("live network test")
	}
	rpcURL := envRPCURL()

	weth := common.HexToAddress("0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2")
	usdt := common.HexToAddress("0xdac17f958d2ee523a2206206994597c13d831ec7")
	engine := common.HexToAddress(fermiEngineAddr)

	overrides, midPrice := fetchTestOverridesForPair(t, weth, usdt)
	t.Logf("WETH/USDT midPrice = %v", midPrice)

	acct, ok := overrides[engine]
	require.True(t, ok)
	t.Logf("engine stateDiff slots: %d", len(acct.StateDiff))

	ec, err := ethclient.Dial(rpcURL)
	require.NoError(t, err)
	defer ec.Close()

	ctx := t.Context()

	fwd, rev := pairKeyForTokens(weth, usdt)
	var matchedKey common.Hash
	for _, pk := range []common.Hash{fwd, rev} {
		base := pairBaseSlot(pk)
		if _, found := acct.StateDiff[slotOffset(base, 1)]; found {
			matchedKey = pk
			break
		}
	}
	if matchedKey == (common.Hash{}) {
		t.Skip("WETH/USDT not in Titan overrides")
	}

	base := pairBaseSlot(matchedKey)
	baseInt := new(big.Int).SetBytes(base[:])

	f0Raw, err := ec.StorageAt(ctx, engine, common.BigToHash(baseInt), nil)
	require.NoError(t, err)
	priceRaw, err := ec.StorageAt(ctx, engine,
		common.BigToHash(new(big.Int).Add(baseInt, big.NewInt(1))), nil)
	require.NoError(t, err)

	onchain := new(big.Int).SetBytes(priceRaw)
	var lub uint64
	if len(f0Raw) == 32 {
		for k := 0; k < 8; k++ {
			lub |= uint64(f0Raw[30-k]) << (8 * k)
		}
	}
	t.Logf("on-chain: lastUpdated=%d midPrice=%s", lub, onchain)

	t5Raw, err := ec.StorageAt(ctx, engine,
		common.BigToHash(new(big.Int).Add(baseInt, big.NewInt(5))), nil)
	require.NoError(t, err)
	t6Raw, err := ec.StorageAt(ctx, engine,
		common.BigToHash(new(big.Int).Add(baseInt, big.NewInt(6))), nil)
	require.NoError(t, err)
	t5 := common.BytesToAddress(t5Raw)
	t6 := common.BytesToAddress(t6Raw)
	t.Logf("tokens: field5=%s field6=%s", t5.Hex(), t6.Hex())
	require.True(t, (t5 == weth && t6 == usdt) || (t5 == usdt && t6 == weth))

	rpcClient := ethrpc.NewWithClient(ec)
	doCall := func(useOverride bool) *big.Int {
		var res struct {
			AmountIn  *big.Int
			AmountOut *big.Int
		}
		req := rpcClient.NewRequest().SetContext(context.Background())
		req.AddCall(&ethrpc.Call{
			ABI:    fermiSwapperABI,
			Target: fermiSwapperAddr,
			Method: methodQuote,
			Params: []any{weth, usdt, big.NewInt(1_000_000_000_000_000_000)},
		}, []any{&res})
		if useOverride {
			req.SetOverrides(overrides)
		}
		_, err := req.Call()
		require.NoError(t, err, "quoteAmounts override=%t", useOverride)
		return res.AmountOut
	}

	plainOut := doCall(false)
	overrideOut := doCall(true)
	t.Logf("WITHOUT override: 1 WETH → %s USDT (raw)", plainOut)
	t.Logf("WITH    override: 1 WETH → %s USDT (raw)", overrideOut)
	if plainOut.Cmp(overrideOut) != 0 {
		t.Logf("drift = %s bps", bpsDrift(plainOut, overrideOut))
	}
}

// ---- helpers ----

func envRPCURL() string {
	if u := os.Getenv("ETH_RPC_URL"); u != "" {
		return u
	}
	return "https://eth.drpc.org"
}

func bpsDrift(a, b *big.Int) string {
	diff := new(big.Int).Sub(a, b)
	diff.Abs(diff)
	max := a
	if b.Cmp(max) > 0 {
		max = b
	}
	drift := new(big.Int).Mul(diff, big.NewInt(10000))
	if max.Sign() > 0 {
		drift.Quo(drift, max)
	}
	return drift.String()
}
