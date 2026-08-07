package flap

import (
	"bytes"
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	commonabi "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// quoteExactInputABI is kept separate from the production portalABI: it's only used here to call
// Portal.quoteExactInput((address,address,uint256)) as an oracle to verify CalcAmountOut against the
// real contract, not by the tracker/simulator/factory. Selector (0xfc847c2b) and the fully static
// 3-word calldata shape (no offset header - QuoteExactInputParams has no dynamic fields) were both
// confirmed live on-chain by decoding a real quoteExactInput() call's calldata layout from the
// decompiled PortalTradeV2 bytecode.
var quoteExactInputABI = func() abi.ABI {
	const abiJSON = `[{
		"inputs": [{
			"components": [
				{"internalType":"address","name":"inputToken","type":"address"},
				{"internalType":"address","name":"outputToken","type":"address"},
				{"internalType":"uint256","name":"amountIn","type":"uint256"}
			],
			"internalType": "struct QuoteExactInputParams",
			"name": "params",
			"type": "tuple"
		}],
		"name": "quoteExactInput",
		"outputs": [{"internalType":"uint256","name":"","type":"uint256"}],
		"stateMutability": "nonpayable",
		"type": "function"
	}]`
	parsed, err := abi.JSON(bytes.NewReader([]byte(abiJSON)))
	if err != nil {
		panic(err)
	}
	return parsed
}()

// TestQuoteDifferential_ExactIn runs the real pipeline - PoolsListUpdater to discover pools,
// PoolTracker to fetch live state, PoolSimulator to price - and compares CalcAmountOut against
// Portal.quoteExactInput on live BSC mainnet, across a range of amounts from small to large in both
// directions. Skipped unless FLAP_RPC_URL and FLAP_API_KEY are set - matches the baseline package's
// quote-differential test convention (see quote_differential_test.go there).
func TestQuoteDifferential_ExactIn(t *testing.T) {
	rpcURL := os.Getenv("FLAP_RPC_URL")
	apiKey := os.Getenv("FLAP_API_KEY")
	if rpcURL == "" || apiKey == "" {
		t.Skip("Set FLAP_RPC_URL and FLAP_API_KEY to run the flap quote differential test")
	}
	apiBaseURL := os.Getenv("FLAP_API_BASE_URL")
	if apiBaseURL == "" {
		apiBaseURL = "https://bnb.taxed.fun"
	}
	portalAddress := os.Getenv("FLAP_PORTAL_ADDRESS")
	if portalAddress == "" {
		portalAddress = "0xe2cE6ab80874Fa9Fa2aAE65D277Dd6B8e65C9De0"
	}

	ctx := context.Background()
	config := &Config{
		DexID:         DexType,
		ChainID:       valueobject.ChainIDBSC,
		PortalAddress: portalAddress,
		APIBaseURL:    apiBaseURL,
		APIKey:        apiKey,
	}

	rpcClient := ethrpc.New(rpcURL).
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	// 1. Discover pools the same way production does.
	lister := NewPoolsListUpdater(config)
	pools, _, err := lister.GetNewPools(ctx, nil)
	require.NoError(t, err)
	require.NotEmpty(t, pools, "graduatinghot board returned no curve-stage pools")

	// The list updater only ever sets Address/Swappable per token (decimals are filled by a
	// downstream enrichment job in production); fill them here the same way before tracking.
	for _, p := range pools {
		for _, tok := range p.Tokens {
			tok.Decimals = fetchDecimals(t, ctx, rpcClient, tok.Address)
		}
	}

	// 2. Track state and simulate with the real tracker/simulator until a still-tradable pool with
	// nonzero circulating supply is found (freshly-launched pools may have near-zero liquidity).
	tracker, err := NewPoolTracker(config, rpcClient)
	require.NoError(t, err)

	var (
		sim         *PoolSimulator
		blockNumber *big.Int
	)
	for _, p := range pools {
		resp, err := rpcClient.NewRequest().SetContext(ctx).TryBlockAndAggregate()
		require.NoError(t, err)
		blockNumber = resp.BlockNumber

		tracked, err := tracker.GetNewPoolState(ctx, p, pool.GetNewPoolStateParams{})
		require.NoError(t, err)

		candidate, err := NewPoolSimulator(tracked)
		require.NoError(t, err)
		if candidate.status != TokenStatusTradable || candidate.circulatingSupply.IsZero() {
			continue
		}
		sim = candidate
		break
	}
	require.NotNil(t, sim, "no trackable, tradable pool found among discovered pools")

	quoteToken := sim.Info.Tokens[0]
	baseToken := sim.Info.Tokens[1]

	amountsInQuoteDecimals := []string{
		"1000000000000000",     // 0.001 quote token
		"10000000000000000",    // 0.01
		"100000000000000000",   // 0.1
		"1000000000000000000",  // 1
		"10000000000000000000", // 10
	}
	for _, amtStr := range amountsInQuoteDecimals {
		t.Run("buy_"+amtStr, func(t *testing.T) {
			amountIn, ok := new(big.Int).SetString(amtStr, 10)
			require.True(t, ok)

			onChainOut := callQuoteExactInput(t, rpcClient, portalAddress, quoteToken, baseToken, amountIn, blockNumber)

			simResult, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: quoteToken, Amount: amountIn},
				TokenOut:      baseToken,
			})
			require.NoError(t, err)

			t.Logf("amountIn=%s onChain=%s simulated=%s", amtStr, onChainOut.String(), simResult.TokenAmountOut.Amount.String())
			require.Equal(t, onChainOut.String(), simResult.TokenAmountOut.Amount.String())
		})
	}

	// Sell direction: a small slice of the token's own circulating supply so it can't underflow.
	sellAmounts := []string{
		"1000000000000000000",    // 1 base token
		"1000000000000000000000", // 1000 base tokens
	}
	for _, amtStr := range sellAmounts {
		t.Run("sell_"+amtStr, func(t *testing.T) {
			amountIn, ok := new(big.Int).SetString(amtStr, 10)
			require.True(t, ok)
			if amountIn.Cmp(sim.circulatingSupply.ToBig()) >= 0 {
				t.Skip("pool's circulating supply too small for this sell amount")
			}

			onChainOut := callQuoteExactInput(t, rpcClient, portalAddress, baseToken, quoteToken, amountIn, blockNumber)

			simResult, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: baseToken, Amount: amountIn},
				TokenOut:      quoteToken,
			})
			require.NoError(t, err)

			t.Logf("amountIn=%s onChain=%s simulated=%s", amtStr, onChainOut.String(), simResult.TokenAmountOut.Amount.String())
			require.Equal(t, onChainOut.String(), simResult.TokenAmountOut.Amount.String())
		})
	}
}

func fetchDecimals(t *testing.T, ctx context.Context, rpcClient *ethrpc.Client, token string) uint8 {
	t.Helper()
	var dec uint8
	_, err := rpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    commonabi.Erc20ABI,
		Target: token,
		Method: "decimals",
	}, []any{&dec}).Call()
	require.NoError(t, err)
	return dec
}

func callQuoteExactInput(t *testing.T, rpcClient *ethrpc.Client, portalAddress, inputToken, outputToken string, amountIn, blockNumber *big.Int) *big.Int {
	t.Helper()
	var out *big.Int
	_, err := rpcClient.NewRequest().SetContext(context.Background()).SetBlockNumber(blockNumber).AddCall(&ethrpc.Call{
		ABI:    quoteExactInputABI,
		Target: portalAddress,
		Method: "quoteExactInput",
		Params: []any{struct {
			InputToken  common.Address
			OutputToken common.Address
			AmountIn    *big.Int
		}{
			InputToken:  common.HexToAddress(inputToken),
			OutputToken: common.HexToAddress(outputToken),
			AmountIn:    amountIn,
		}},
	}, []any{&out}).Call()
	require.NoError(t, err)
	return out
}
