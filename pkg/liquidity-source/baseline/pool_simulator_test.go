package baseline

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

const (
	mainnetRelay = "0xc81Fd894C0acE037d133aF4886550aC8133568E8"
	mainnetWETH  = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
	mainnetB     = "0x9fdbde76236998dc2836fe67a9954ede456a1d63"
)

func skipIfNoRPC(t *testing.T) string {
	t.Helper()
	rpcURL := os.Getenv("ETH_RPC_URL")
	if rpcURL == "" {
		t.Skip("Set ETH_RPC_URL to run live tests")
	}
	return rpcURL
}

func skipIfNoSubgraph(t *testing.T) (rpcURL, graphqlURL, relayAddr string) {
	t.Helper()
	rpcURL = os.Getenv("BASELINE_RPC_URL")
	graphqlURL = os.Getenv("BASELINE_GRAPHQL_URL")
	relayAddr = os.Getenv("BASELINE_RELAY_ADDRESS")
	if rpcURL == "" || graphqlURL == "" || relayAddr == "" {
		t.Skip("Set BASELINE_RPC_URL, BASELINE_GRAPHQL_URL, and BASELINE_RELAY_ADDRESS to run subgraph tests")
	}
	return
}

func newMainnetPool() entity.Pool {
	return entity.Pool{
		Address:  mainnetB,
		Exchange: "baseline",
		Type:     DexType,
		Reserves: entity.PoolReserves{"0", "0"},
		Tokens: []*entity.PoolToken{
			{Address: mainnetWETH, Decimals: 18, Symbol: "WETH", Swappable: true},
			{Address: mainnetB, Decimals: 18, Symbol: "B", Swappable: true},
		},
	}
}

func TestPoolsListUpdater_GetNewPools(t *testing.T) {
	rpcURL, graphqlURL, relayAddr := skipIfNoSubgraph(t)

	ethrpcClient := ethrpc.New(rpcURL)
	ethrpcClient.SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	graphqlClient := graphqlpkg.NewClient(graphqlURL)

	cfg := &Config{
		DexID:        "baseline",
		ChainID:      1,
		RelayAddress: relayAddr,
		NewPoolLimit: 10,
	}

	updater := NewPoolsListUpdater(cfg, ethrpcClient, graphqlClient)

	pools, metadata, err := updater.GetNewPools(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetNewPools failed: %v", err)
	}

	t.Logf("Found %d pools", len(pools))
	for _, p := range pools {
		t.Logf("  Pool: %s (%s/%s)", p.Address, p.Tokens[0].Symbol, p.Tokens[1].Symbol)
	}
	t.Logf("Metadata: %s", string(metadata))
}

func TestPoolTracker_GetNewPoolState(t *testing.T) {
	rpcURL := skipIfNoRPC(t)

	ethrpcClient := ethrpc.New(rpcURL)
	ethrpcClient.SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	cfg := &Config{
		DexID:        "baseline",
		ChainID:      1,
		RelayAddress: mainnetRelay,
	}

	tracker, err := NewPoolTracker(cfg, ethrpcClient)
	if err != nil {
		t.Fatalf("NewPoolTracker failed: %v", err)
	}

	updated, err := tracker.GetNewPoolState(context.Background(), newMainnetPool(), pool.GetNewPoolStateParams{})
	if err != nil {
		t.Fatalf("GetNewPoolState failed: %v", err)
	}

	t.Logf("Reserves: %v", updated.Reserves)

	var extra Extra
	if err := json.Unmarshal([]byte(updated.Extra), &extra); err != nil {
		t.Fatalf("Failed to unmarshal extra: %v", err)
	}

	if extra.QuoteState == nil {
		t.Fatal("QuoteState not populated")
	}
	if extra.QuoteState.SnapshotCurveParams.BLV == nil || extra.QuoteState.SnapshotCurveParams.BLV.IsZero() {
		t.Fatal("QuoteState snapshot BLV not populated")
	}

	t.Logf("Quote state: total reserves=%s total bTokens=%s", extra.QuoteState.TotalReserves, extra.QuoteState.TotalBTokens)
}

func TestCalcAmountOut_BuyPopulatesAmountOut(t *testing.T) {
	rpcURL := skipIfNoRPC(t)

	ethrpcClient := ethrpc.New(rpcURL)
	ethrpcClient.SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	cfg := &Config{
		DexID:        "baseline",
		ChainID:      1,
		RelayAddress: mainnetRelay,
	}

	tracker, err := NewPoolTracker(cfg, ethrpcClient)
	if err != nil {
		t.Fatalf("NewPoolTracker failed: %v", err)
	}

	updated, err := tracker.GetNewPoolState(context.Background(), newMainnetPool(), pool.GetNewPoolStateParams{})
	if err != nil {
		t.Fatalf("GetNewPoolState failed: %v", err)
	}

	sim, err := NewPoolSimulator(updated)
	if err != nil {
		t.Fatalf("NewPoolSimulator failed: %v", err)
	}

	// Buy: 0.01 WETH -> $B
	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  mainnetWETH,
			Amount: big.NewInt(1e16),
		},
		TokenOut: mainnetB,
	})
	if err != nil {
		t.Fatalf("CalcAmountOut (buy) failed: %v", err)
	}

	swapInfo, ok := result.SwapInfo.(SwapInfo)
	if !ok {
		t.Fatal("SwapInfo is not of type SwapInfo")
	}

	if !swapInfo.IsBuy {
		t.Fatal("expected IsBuy to be true for reserve -> bToken")
	}

	if swapInfo.AmountOut == "" {
		t.Fatal("SwapInfo.AmountOut must be populated for buys (needed for efficient buyTokensExactOut path)")
	}

	amountOut, ok := new(big.Int).SetString(swapInfo.AmountOut, 10)
	if !ok {
		t.Fatalf("SwapInfo.AmountOut is not a valid decimal string: %q", swapInfo.AmountOut)
	}

	if amountOut.Cmp(result.TokenAmountOut.Amount) != 0 {
		t.Fatalf("SwapInfo.AmountOut (%s) != TokenAmountOut.Amount (%s)", amountOut, result.TokenAmountOut.Amount)
	}

	t.Logf("Buy: 0.01 WETH -> %s $B (AmountOut in SwapInfo: %s)", result.TokenAmountOut.Amount, swapInfo.AmountOut)
}

func TestCalcAmountOut_SellOmitsAmountOut(t *testing.T) {
	rpcURL := skipIfNoRPC(t)

	ethrpcClient := ethrpc.New(rpcURL)
	ethrpcClient.SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	cfg := &Config{
		DexID:        "baseline",
		ChainID:      1,
		RelayAddress: mainnetRelay,
	}

	tracker, err := NewPoolTracker(cfg, ethrpcClient)
	if err != nil {
		t.Fatalf("NewPoolTracker failed: %v", err)
	}

	updated, err := tracker.GetNewPoolState(context.Background(), newMainnetPool(), pool.GetNewPoolStateParams{})
	if err != nil {
		t.Fatalf("GetNewPoolState failed: %v", err)
	}

	sim, err := NewPoolSimulator(updated)
	if err != nil {
		t.Fatalf("NewPoolSimulator failed: %v", err)
	}

	// Sell: 1000 $B -> WETH
	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  mainnetB,
			Amount: new(big.Int).Mul(big.NewInt(1000), big.NewInt(1e18)),
		},
		TokenOut: mainnetWETH,
	})
	if err != nil {
		t.Fatalf("CalcAmountOut (sell) failed: %v", err)
	}

	swapInfo, ok := result.SwapInfo.(SwapInfo)
	if !ok {
		t.Fatal("SwapInfo is not of type SwapInfo")
	}

	if swapInfo.IsBuy {
		t.Fatal("expected IsBuy to be false for bToken -> reserve")
	}

	if swapInfo.AmountOut != "" {
		t.Fatalf("SwapInfo.AmountOut should be empty for sells, got: %s", swapInfo.AmountOut)
	}

	t.Logf("Sell: 1000 $B -> %s WETH (AmountOut correctly omitted)", result.TokenAmountOut.Amount)
}
