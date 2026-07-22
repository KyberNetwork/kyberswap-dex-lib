package machima

import (
	"bytes"
	"context"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/goccy/go-json"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Deployed Machima contracts on Base, aggregator v1.1.0: residual refund to recipient, per-token
// sell floors in the quoter, native ETH entry points, and a swapAvailability(token) anti-sniper view.
const (
	aggregatorQuoter = "0xafB47806e61c9888Eb4A1047BfBf59C29680B8e4"
	aggregatorRouter = "0xa25D1158B7Cf373DC3787793A52933dB0A0CaD89"
	clankNowAddr     = "0x44FefF82302D231dcC30f97280D1c9843F308D1a"
	tickLensAddr     = "0x3FAD85D470f87e4fa615a9dA06032c0E264D4DF4"
	swapAdapterAddr  = "0x9FFB6a12d14b0F86AC122486081e3B86728E65F9"

	multicall3 = "0xcA11bde05977b3631167028862bE2a173976CA11"
)

// MachimaAggregatorQuoter.quote is non-view (QuoterV2's revert pattern), so it cannot go through
// Multicall3 and is called via a direct eth_call.
const quoterABIJSON = `[{
	"inputs": [
		{"internalType": "address", "name": "tokenIn", "type": "address"},
		{"internalType": "address", "name": "tokenOut", "type": "address"},
		{"internalType": "uint256", "name": "amountIn", "type": "uint256"}
	],
	"name": "quote",
	"outputs": [
		{"internalType": "uint256", "name": "amountOut", "type": "uint256"},
		{"internalType": "uint256", "name": "taxAmount", "type": "uint256"},
		{"internalType": "uint16", "name": "taxBps", "type": "uint16"}
	],
	"stateMutability": "nonpayable",
	"type": "function"
}]`

// TestPoolSimulatorParityWithQuoter is the evidence that the whole port is right: it tracks a live
// pool and asserts the simulator matches MachimaAggregatorQuoter.quote() wei-for-wei.
//
// It is also what settles the protocolTaxBps* question documented on TaxConfig — the quoter applies
// every deduction the router does, so a matching output means buy/sell tax alone is the full story.
//
// Requires BASE_RPC_URL; the deterministic tests in pool_simulator_test.go are what run in CI.
func TestPoolSimulatorParityWithQuoter(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping fork test in short mode")
	}
	rpcURL := os.Getenv("BASE_RPC_URL")
	if rpcURL == "" {
		t.Skip("BASE_RPC_URL not set (public Base RPC rate limits are too aggressive)")
	}

	ctx := context.Background()
	// Built through the real constructor so the delegated UniV3 tracker is wired too. The GraphQL
	// client is nil on purpose: AlwaysUseTickLens means ticks come from the TickLens contract.
	tracker, err := NewPoolTracker(&Config{
		DexID:           DexType,
		ClankNow:        clankNowAddr,
		SwapAdapter:     swapAdapterAddr,
		TickLensAddress: tickLensAddr,
		RouterAddress:   aggregatorRouter,
		WETH:            wethAddr,
		USDC:            usdcAddr,
		XMA:             xmaAddr,
	}, ethrpc.New(rpcURL).SetMulticallContract(common.HexToAddress(multicall3)), nil)
	require.NoError(t, err)

	// XMA/WETH: XMA is the traded token even though it is also a counter asset, which is the
	// both-counter case _classifyPair exists for.
	tests := []struct {
		name                string
		poolAddress         string
		token0, token1      string
		token, counterAsset string
		buySizes, sellSizes []*big.Int
	}{
		{
			name:         "XMA/WETH",
			poolAddress:  "0x531aae7d71343c663821604c57520b1602567006",
			token0:       wethAddr,
			token1:       xmaAddr,
			token:        xmaAddr,
			counterAsset: wethAddr,
			buySizes: []*big.Int{
				big.NewInt(1e14), // 0.0001 ETH
				big.NewInt(1e15), // 0.001 ETH
				big.NewInt(5e15), // 0.005 ETH
			},
			sellSizes: []*big.Int{
				new(big.Int).Mul(big.NewInt(1e18), big.NewInt(10)),
				new(big.Int).Mul(big.NewInt(1e18), big.NewInt(100)),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			staticExtra, err := json.Marshal(StaticExtra{
				Token:         tc.token,
				RouterAddress: aggregatorRouter,
				WETH:          wethAddr,
				USDC:          usdcAddr,
				XMA:           xmaAddr,
			})
			require.NoError(t, err)

			// tickSpacing seed, exactly as PoolsListUpdater writes it: ticklens reads it out of
			// Extra to know which tick words to scan.
			seedExtra, err := json.Marshal(Extra{Extra: uniswapv3.Extra{TickSpacing: defaultTickSpacing}})
			require.NoError(t, err)

			tracked, err := tracker.BootstrapPoolState(ctx, entity.Pool{
				Address:  tc.poolAddress,
				SwapFee:  defaultFee,
				Exchange: DexType,
				Type:     DexType,
				Tokens: []*entity.PoolToken{
					{Address: tc.token0, Swappable: true},
					{Address: tc.token1, Swappable: true},
				},
				Reserves:    entity.PoolReserves{"0", "0"},
				Extra:       string(seedExtra),
				StaticExtra: string(staticExtra),
			}, pool.GetNewPoolStateParams{})
			require.NoError(t, err, "bootstrap should fetch pool state")

			// The interval trigger (no logs) must refresh state without dropping the ticks the
			// bootstrap fetched — that is what keeps tax and the XMA floor fresh.
			refreshed, err := tracker.GetNewPoolState(ctx, tracked, pool.GetNewPoolStateParams{})
			require.NoError(t, err, "interval refresh should succeed")
			var beforeExtra, afterExtra Extra
			require.NoError(t, json.Unmarshal([]byte(tracked.Extra), &beforeExtra))
			require.NoError(t, json.Unmarshal([]byte(refreshed.Extra), &afterExtra))
			require.NotEmpty(t, beforeExtra.Ticks, "bootstrap should have fetched ticks")
			assert.Len(t, afterExtra.Ticks, len(beforeExtra.Ticks), "interval refresh must keep ticks")
			assert.Equal(t, beforeExtra.HasTax, afterExtra.HasTax)
			t.Logf("bootstrap: ticks=%d tick=%v hasTax=%v buy=%d sell=%d floor=%v",
				len(beforeExtra.Ticks), beforeExtra.Tick, beforeExtra.HasTax,
				beforeExtra.BuyTaxBps, beforeExtra.SellTaxBps, beforeExtra.XmaSellSqrtPriceLimit)
			// FetchPoolTicks re-reads Extra as a uniswapv3.Extra. This is the call that failed in
			// pool-service bootstrap when the shared fields were uint256-typed.
			refetched, err := tracker.FetchPoolTicks(ctx, refreshed)
			require.NoError(t, err, "FetchPoolTicks must be able to read the Extra we wrote")
			var refetchedExtra Extra
			require.NoError(t, json.Unmarshal([]byte(refetched.Extra), &refetchedExtra))
			assert.Len(t, refetchedExtra.Ticks, len(beforeExtra.Ticks))
			// This is the state that actually lands in Redis after bootstrap. Losing the tax here
			// is silent: the pool would just quote untaxed.
			assert.Equal(t, beforeExtra.ProtocolState, refetchedExtra.ProtocolState,
				"FetchPoolTicks must not drop the Machima half of Extra")
			assert.True(t, refetchedExtra.HasTax, "XMA has a tax configured on-chain")

			tracked = refreshed

			sim, err := NewPoolSimulator(tracked, valueobject.ChainIDBase)
			require.NoError(t, err)

			assertParity := func(t *testing.T, tokenIn, tokenOut string, amountIn *big.Int) {
				t.Helper()
				time.Sleep(time.Second) // public RPC rate limits

				got, simErr := sim.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amountIn},
					TokenOut:      tokenOut,
				})
				want, quoterErr := quote(rpcURL, tokenIn, tokenOut, amountIn)

				if simErr != nil {
					// Refusing to quote is only correct if the pool would also revert on-chain,
					// e.g. when the price is pinned at the XMA sell floor.
					require.Error(t, quoterErr,
						"simulator refused (%v) but the on-chain quoter succeeded — real parity break", simErr)
					t.Logf("both refuse: sim=%v quoter=%v", simErr, quoterErr)
					return
				}
				require.NoError(t, quoterErr)

				// Allow the larger of 100 wei or 1bp; V3 fixed-point math diverges by a few wei.
				diff := new(big.Int).Sub(got.TokenAmountOut.Amount, want)
				diff.Abs(diff)
				tolerance := new(big.Int).Div(want, big.NewInt(10000))
				if tolerance.Cmp(big.NewInt(100)) < 0 {
					tolerance = big.NewInt(100)
				}
				assert.LessOrEqual(t, diff.Cmp(tolerance), 0,
					"sim=%s quoter=%s diff=%s tolerance=%s", got.TokenAmountOut.Amount, want, diff, tolerance)
			}

			for _, amountIn := range tc.buySizes {
				t.Run("buy_"+amountIn.String(), func(t *testing.T) {
					assertParity(t, tc.counterAsset, tc.token, amountIn)
				})
			}
			for _, amountIn := range tc.sellSizes {
				t.Run("sell_"+amountIn.String(), func(t *testing.T) {
					assertParity(t, tc.token, tc.counterAsset, amountIn)
				})
			}

			// A sell large enough to reach the launch-tick floor: the floor coincides with the last
			// initialized tick, so the simulator cannot compute the partial fill and skips the pool.
			// The on-chain quoter reverts in the same state, which assertParity checks.
			t.Run("sell_at_floor", func(t *testing.T) {
				assertParity(t, tc.token, tc.counterAsset,
					new(big.Int).Mul(big.NewInt(1e18), big.NewInt(5000)))
			})
		})
	}
}

// quote calls MachimaAggregatorQuoter.quote, returning the revert as an error so callers can assert
// error parity. Transient rate limits are retried so they cannot masquerade as quoter reverts.
func quote(rpcURL, tokenIn, tokenOut string, amountIn *big.Int) (*big.Int, error) {
	quoterABI, err := abi.JSON(bytes.NewReader([]byte(quoterABIJSON)))
	if err != nil {
		return nil, err
	}
	calldata, err := quoterABI.Pack("quote",
		common.HexToAddress(tokenIn), common.HexToAddress(tokenOut), amountIn)
	if err != nil {
		return nil, err
	}

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	to := common.HexToAddress(aggregatorQuoter)
	var result []byte
	for attempt := 0; ; attempt++ {
		result, err = client.CallContract(context.Background(),
			ethereum.CallMsg{To: &to, Data: calldata}, nil)
		if err == nil {
			break
		}
		if attempt >= 4 || !strings.Contains(err.Error(), "rate limit") {
			return nil, err
		}
		time.Sleep(time.Duration(attempt+1) * 2 * time.Second)
	}

	outputs, err := quoterABI.Unpack("quote", result)
	if err != nil {
		return nil, err
	}
	if len(outputs) == 0 {
		return nil, errors.New("quoter returned no outputs")
	}
	amountOut, ok := outputs[0].(*big.Int)
	if !ok {
		return nil, errors.Errorf("unexpected amountOut type %T", outputs[0])
	}
	return amountOut, nil
}
