package ringswapbacking

import (
	"context"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
)

const (
	multicall3Mainnet = "0xcA11bde05977b3631167028862bE2a173976CA11"
	verifyWETH        = "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"
	verifyUSDT        = "0xdAC17F958D2ee523a2206206994597C13D831ec7"
)

// TestVerifyLocalForkPipeline covers Kyber's lister -> tracker -> simulator path against an
// actual FewBackingAwareV2Router deployed on a local mainnet fork. The Ring runner funds a real
// Euler-backed Manager, revokes its minter role, and keeps this opt-in test out of ordinary runs.
func TestVerifyLocalForkPipeline(t *testing.T) {
	rpcURL := os.Getenv("RINGSWAP_BACKING_VERIFY_RPC_URL")
	routerAddress := os.Getenv("RINGSWAP_BACKING_VERIFY_ROUTER")
	if rpcURL == "" || routerAddress == "" {
		t.Skip("local fork verification environment is not set")
	}
	require.True(t, common.IsHexAddress(routerAddress), "router address")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client := ethrpc.New(rpcURL).SetMulticallContract(common.HexToAddress(multicall3Mainnet))
	config := &Config{
		DexID: DexType,
		Routers: []RouterConfig{{
			Address:             routerAddress,
			ReplaceOrdinaryPair: true,
			NoRecallGasToken0:   178_435,
			NoRecallGasToken1:   181_377,
			RecallGasToken0:     399_312,
			RecallGasToken1:     378_580,
		}},
	}
	lister := NewPoolsListUpdater(config, client)

	pools, metadata, err := lister.GetNewPools(ctx, nil)
	require.NoError(t, err)
	require.Len(t, pools, 1)
	require.Equal(t, strings.ToLower(verifyWETH), pools[0].Tokens[0].Address)
	require.Equal(t, strings.ToLower(verifyUSDT), pools[0].Tokens[1].Address)
	var staticExtra StaticExtra
	require.NoError(t, json.Unmarshal([]byte(pools[0].StaticExtra), &staticExtra))
	require.Equal(t, staticExtra.PairAddress, pools[0].Address)
	require.Equal(t, strings.ToLower(routerAddress), staticExtra.RouterAddress)

	var persisted PoolsListUpdaterMetadata
	require.NoError(t, json.Unmarshal(metadata, &persisted))
	require.Equal(t, []string{strings.ToLower(routerAddress)}, persisted.KnownRouters)
	require.Equal(t, []string{staticExtra.PairAddress}, persisted.KnownPairs)
	secondPools, secondMetadata, err := lister.GetNewPools(ctx, metadata)
	require.NoError(t, err)
	require.Empty(t, secondPools, "persisted routers must not be rediscovered")
	require.Equal(t, metadata, secondMetadata)

	tracked, err := NewPoolTracker(config, client).GetNewPoolState(
		ctx, pools[0], pool.GetNewPoolStateParams{},
	)
	require.NoError(t, err)
	require.Positive(t, tracked.BlockNumber)
	extra := decodeExtra(t, tracked.Extra)
	require.Positive(t, extra.RecallCapacity1.Sign(), "USDT recall capacity")

	simulator, err := NewPoolSimulator(tracked)
	require.NoError(t, err)
	limit := swaplimit.NewInventory(DexType, simulator.CalculateLimit())
	hotResult, err := simulator.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token: verifyWETH, Amount: big.NewInt(100_000_000_000_000_000),
		},
		TokenOut: verifyUSDT,
		Limit:    limit,
	})
	require.NoError(t, err)
	require.False(t, hotResult.SwapInfo.(SwapInfo).UseRecall)
	require.Equal(t, int64(181_377), hotResult.Gas)

	verifyRecallQuote(
		t,
		ctx,
		client,
		simulator,
		verifyWETH,
		verifyUSDT,
		recallTriggerAmountIn(t, tracked.Reserves[0], tracked.Reserves[1], extra),
		limit,
	)
}

func recallTriggerAmountIn(
	t *testing.T,
	reserveInRaw string,
	reserveOutRaw string,
	extra Extra,
) *big.Int {
	t.Helper()
	reserveIn, ok := new(big.Int).SetString(reserveInRaw, 10)
	require.True(t, ok, "reserve in")
	reserveOut, ok := new(big.Int).SetString(reserveOutRaw, 10)
	require.True(t, ok, "reserve out")
	require.Positive(t, extra.RecallCapacity1.Sign(), "output recall capacity")

	headroom := new(big.Int).Sub(reserveOut, extra.WrapperBuffer1)
	require.Positive(t, headroom.Sign(), "Pair output reserve must exceed hot wrapper backing")
	recall := new(big.Int).Div(new(big.Int).Set(extra.RecallCapacity1), big.NewInt(2))
	halfHeadroom := new(big.Int).Div(new(big.Int).Set(headroom), big.NewInt(2))
	if recall.Cmp(halfHeadroom) > 0 {
		recall = halfHeadroom
	}
	if recall.Sign() == 0 {
		recall.SetInt64(1)
	}
	targetOut := new(big.Int).Add(extra.WrapperBuffer1, recall)

	// Exact inverse of the Pair's 997/1000 formula, rounded up like Uniswap v2 getAmountIn.
	numerator := new(big.Int).Mul(reserveIn, targetOut)
	numerator.Mul(numerator, big.NewInt(1_000))
	denominator := new(big.Int).Sub(reserveOut, targetOut)
	denominator.Mul(denominator, big.NewInt(997))
	require.Positive(t, denominator.Sign(), "recall target below Pair reserve")
	return new(big.Int).Add(new(big.Int).Div(numerator, denominator), big.NewInt(1))
}

func verifyRecallQuote(
	t *testing.T,
	ctx context.Context,
	client *ethrpc.Client,
	simulator *PoolSimulator,
	tokenIn string,
	tokenOut string,
	amountIn *big.Int,
	limit pool.SwapLimit,
) {
	t.Helper()
	result, err := simulator.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amountIn},
		TokenOut:      tokenOut,
		Limit:         limit,
	})
	require.NoError(t, err)
	require.Equal(t, int64(378_580), result.Gas)
	require.True(t, result.SwapInfo.(SwapInfo).UseRecall)

	var chainResult routeQuoteResult
	_, err = client.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI: routerABI, Target: simulator.RouterAddress, Method: "routeQuote",
		Params: []any{common.HexToAddress(tokenIn), amountIn},
	}, []any{&chainResult}).TryBlockAndAggregate()
	require.NoError(t, err)
	chainQuote := chainResult.Quote
	require.NotNil(t, chainQuote.AmountOut)
	require.True(t, chainQuote.RecallRequired)
	require.True(t, chainQuote.Executable)
	require.Equal(t, chainQuote.AmountOut, result.TokenAmountOut.Amount)
}

func decodeExtra(t *testing.T, raw string) Extra {
	t.Helper()
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(raw), &extra))
	return extra
}
