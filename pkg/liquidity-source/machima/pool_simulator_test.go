package machima

import (
	"context"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (


	// Deployed contracts on Base
	aggregatorQuoter = "0x9dA94300DEC6ac282880f71df3270a922Bcbd034"
	aggregatorRouter = "0x566250347E1401615B3e043918fc290B98448578"
	clankNowAddr     = "0x44FefF82302D231dcC30f97280D1c9843F308D1a"
	tickLensAddr     = "0x3FAD85D470f87e4fa615a9dA06032c0E264D4DF4"

	wethAddr = "0x4200000000000000000000000000000000000006"
	usdcAddr = "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"
	xmaAddr  = "0xa4985faeb1e64ba215282255dbb78ff59c63d7a9"
)

// quoterABI for MachimaAggregatorQuoter.quote(address,address,uint256)
// returns (uint256 amountOut, uint256 taxAmount, uint16 taxBps)
var quoterABIJSON = `[{
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

// TestPoolSimulatorParityWithQuoter forks Base and asserts that the Go
// PoolSimulator produces the same amountOut as the on-chain
// MachimaAggregatorQuoter.quote() for various pool/direction/size combos.
//
// This is the definitive correctness test: if the simulator matches the
// quoter wei-for-wei (or within rounding tolerance), we know the tax math,
// tick-crossing logic, and fee handling are all correct.
func TestPoolSimulatorParityWithQuoter(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping fork test in short mode")
	}

	ctx := context.Background()

	rpcURL := os.Getenv("BASE_RPC_URL")
	if rpcURL == "" {
		t.Skip("BASE_RPC_URL not set — skipping parity test (public RPC rate limits too aggressive)")
	}

	ethrpcClient := ethrpc.New(rpcURL).
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	// Build tracker config
	cfg := &Config{
		DexID:           DexTypeMachima,
		ClankNow:        clankNowAddr,
		TickLensAddress: tickLensAddr,
		RouterAddress:   aggregatorRouter,
		WETH:            wethAddr,
		USDC:            usdcAddr,
		XMA:             xmaAddr,
	}

	tracker := &PoolTracker{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}

	// Test cases: real pools on Base
	// TODO: fill poolAddress with actual active pools from the subgraph
	type testPool struct {
		name         string
		poolAddress  string
		token0       string // on-chain token0
		token1       string // on-chain token1
		token        string // launched token (for tax lookup)
		counterAsset string // WETH/USDC/XMA
	}

	// XMA/WETH — XMA is the traded token, WETH is the counter-asset.
	// The redeployed quoter uses _classifyPair to handle this both-counter case.
	pools := []testPool{
		{
			name:         "XMA/WETH",
			poolAddress:  "0x531aae7d71343c663821604c57520b1602567006",
			token0:       wethAddr, // WETH is token0
			token1:       xmaAddr,  // XMA is token1
			token:        xmaAddr,
			counterAsset: wethAddr,
		},
	}

	// Sizes to test — keep within pool's available liquidity
	buySizes := []*big.Int{
		big.NewInt(1e14),   // 0.0001 ETH (small)
		big.NewInt(1e15),   // 0.001 ETH (medium)
		big.NewInt(5e15),   // 0.005 ETH (larger)
	}
	sellSizes := []*big.Int{
		new(big.Int).Mul(big.NewInt(1e18), big.NewInt(10)),  // 10 XMA (small)
		new(big.Int).Mul(big.NewInt(1e18), big.NewInt(100)), // 100 XMA (medium, within liquidity range)
	}
	// Large sell that hits the floor/liquidity edge — tested separately below
	floorSellSize := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(5000))

	for _, tp := range pools {
		t.Run(tp.name, func(t *testing.T) {
			// Build entity.Pool for the tracker.
			// Token ordering must match the on-chain pool (token0, token1).
			ep := entity.Pool{
				Address:  tp.poolAddress,
				SwapFee:  float64(PoolFee),
				Exchange: DexTypeMachima,
				Type:     DexTypeMachima,
				Tokens: []*entity.PoolToken{
					{Address: tp.token0, Swappable: true},
					{Address: tp.token1, Swappable: true},
				},
				Reserves:    []string{"0", "0"},
				StaticExtra: mustJSON(StaticExtra{
					CounterAsset:  tp.counterAsset,
					Token:         tp.token,
					RouterAddress: aggregatorRouter,
					WETH:          wethAddr,
					USDC:          usdcAddr,
					XMA:           xmaAddr,
				}),
			}

			// Track pool state (fetches slot0, ticks, tax, reserves)
			tracked, err := tracker.GetNewPoolState(ctx, ep, pool.GetNewPoolStateParams{})
			require.NoError(t, err, "tracker should fetch pool state")

			// Build simulator
			sim, err := NewPoolSimulator(tracked, valueobject.ChainID(8453))
			require.NoError(t, err, "simulator should initialize")

			// --- BUY tests (counterAsset → token) ---
			for i, amountIn := range buySizes {
				t.Run("buy_"+bigStr(amountIn), func(t *testing.T) {
					time.Sleep(1 * time.Second) // rate limit avoidance
					_ = i

					// Simulator
					simResult, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
						TokenAmountIn: pool.TokenAmount{Token: tp.counterAsset, Amount: amountIn},
						TokenOut:      tp.token,
					})
					require.NoError(t, err, "sim buy should not error")

					// On-chain quoter
					onChainOut := callQuoter(t, rpcURL, tp.counterAsset, tp.token, amountIn)

					// Assert parity: allow up to 100 wei absolute or 1bps relative
					// (V3 fixed-point math can diverge by a few wei on large outputs)
					diff := new(big.Int).Sub(simResult.TokenAmountOut.Amount, onChainOut)
					diff.Abs(diff)
					tolerance := big.NewInt(100)
					// relative: 1bps of on-chain output
					relTol := new(big.Int).Div(onChainOut, big.NewInt(10000))
					if relTol.Cmp(tolerance) > 0 {
						tolerance = relTol
					}
					assert.True(t, diff.Cmp(tolerance) <= 0,
						"buy parity: sim=%s on-chain=%s diff=%s tol=%s",
						simResult.TokenAmountOut.Amount, onChainOut, diff, tolerance)
				})
			}

			// --- SELL tests (token → counterAsset) ---
			for i, amountIn := range sellSizes {
				t.Run("sell_"+bigStr(amountIn), func(t *testing.T) {
					time.Sleep(1 * time.Second) // rate limit avoidance
					_ = i

					simResult, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
						TokenAmountIn: pool.TokenAmount{Token: tp.token, Amount: amountIn},
						TokenOut:      tp.counterAsset,
					})
					require.NoError(t, err, "sim sell should not error")

					onChainOut := callQuoter(t, rpcURL, tp.token, tp.counterAsset, amountIn)

					diff := new(big.Int).Sub(simResult.TokenAmountOut.Amount, onChainOut)
					diff.Abs(diff)
					tolerance := big.NewInt(100)
					relTol := new(big.Int).Div(onChainOut, big.NewInt(10000))
					if relTol.Cmp(tolerance) > 0 {
						tolerance = relTol
					}
					assert.True(t, diff.Cmp(tolerance) <= 0,
						"sell parity: sim=%s on-chain=%s diff=%s tol=%s",
						simResult.TokenAmountOut.Amount, onChainOut, diff, tolerance)
				})
			}

			// --- FLOOR-EDGE test: sell large enough to hit xmaSellSqrtPriceLimit ---
			// The floor coincides with the liquidity edge (last initialized tick).
			// The Kyber UniV3 sim cannot compute a partial fill at/beyond the last tick —
			// it returns ErrAtOrAboveLargest. This is correct behavior: Kyber skips
			// this pool for whale-sized sells that would only partially fill on-chain.
			t.Run("sell_floor_edge_"+bigStr(floorSellSize), func(t *testing.T) {
				_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: tp.token, Amount: floorSellSize},
					TokenOut:      tp.counterAsset,
				})
				assert.Error(t, err, "floor-edge sell should error (sim cannot quote at liquidity boundary)")
				t.Logf("floor-edge sell correctly errors: %v", err)
			})
		})
	}
}

// TestCalcAmountOut_ErrInvalidPair verifies pair rejection scenarios.
func TestCalcAmountOut_ErrInvalidPair(t *testing.T) {
	t.Run("neither_side_is_counter", func(t *testing.T) {
		sim := &PoolSimulator{}
		_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: "0xaaaa", Amount: big.NewInt(1e18)},
			TokenOut:      "0xbbbb",
		})
		assert.ErrorIs(t, err, ErrInvalidPair)
	})

	t.Run("both_external_counters_WETH_USDC", func(t *testing.T) {
		sim := &PoolSimulator{
			WETH: wethAddr,
			USDC: usdcAddr,
			XMA:  xmaAddr,
		}
		_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: wethAddr, Amount: big.NewInt(1e18)},
			TokenOut:      usdcAddr,
		})
		assert.ErrorIs(t, err, ErrInvalidPair)
	})
}

// TestCalcAmountOut_ErrAntiSniperActive verifies the anti-sniper window.
func TestCalcAmountOut_ErrAntiSniperActive(t *testing.T) {
	sim := &PoolSimulator{
		CounterAsset:       wethAddr,
		WETH:               wethAddr,
		USDC:               usdcAddr,
		XMA:                xmaAddr,
		PoolDeploymentTime: 9999999999, // far future → window active
	}

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: wethAddr, Amount: big.NewInt(1e18)},
		TokenOut:      "0x1234567890abcdef1234567890abcdef12345678",
	})
	assert.ErrorIs(t, err, ErrAntiSniperActive)
}

// callQuoter calls MachimaAggregatorQuoter.quote() on-chain via a direct eth_call.
// quote() is non-view (QuoterV2 uses state-modifying revert pattern) so it cannot
// go through Multicall3. We use ethclient.CallContract directly.
func callQuoter(t *testing.T, rpcURL, tokenIn, tokenOut string, amountIn *big.Int) *big.Int {
	t.Helper()

	qABI := mustParseABI(quoterABIJSON)

	calldata, err := qABI.Pack("quote",
		common.HexToAddress(tokenIn),
		common.HexToAddress(tokenOut),
		amountIn,
	)
	require.NoError(t, err, "pack quote calldata")

	client, err := ethclient.Dial(rpcURL)
	require.NoError(t, err, "dial ethclient")
	defer client.Close()

	to := common.HexToAddress(aggregatorQuoter)
	result, err := client.CallContract(context.Background(), ethereum.CallMsg{
		To:   &to,
		Data: calldata,
	}, nil)
	require.NoError(t, err, "eth_call to quoter should succeed")

	outputs, err := qABI.Unpack("quote", result)
	require.NoError(t, err, "unpack quote result")
	require.True(t, len(outputs) >= 1, "expected at least 1 output")

	amountOut, ok := outputs[0].(*big.Int)
	require.True(t, ok, "first output should be *big.Int")
	return amountOut
}

func mustJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func bigStr(b *big.Int) string {
	return b.String()
}

// Suppress unused import warnings for test helpers
var _ = uint256.NewInt
var _ = valueobject.ChainIDBase
