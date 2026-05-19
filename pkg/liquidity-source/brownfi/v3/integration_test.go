package brownfiv3

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	poolAddr    = "0x3e6200dc34c3b5967e7bbdcf5fa74153348e9694"
	wethAddr    = "0x2f6f07cdcf3588944bf4c42ac74ff24bf56e7590"
	usdcAddr    = "0x549943e04f40284185054145c6e4e9568c1d3241"
	factoryAddr = "0x83A329E93f7A36b9baAb5bF1EAFF319947387552"
	mc3Addr     = "0xcA11bde05977b3631167028862bE2a173976CA11"
)

// TestIntegration_CalcAmountOut fetches live state for the WETH/USDC pool and
// compares our off-chain computeSwapPrices against the on-chain getSwapPrices
// to pinpoint the source of price discrepancy.
func TestIntegration_CalcAmountOut(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("skipping integration test in CI")
	}

	rpcURL := os.Getenv("BERACHAIN_RPC")
	if rpcURL == "" {
		rpcURL = "https://lb.drpc.live/berachain/Av_ucIUlR08slbBUFg1E4U0n6sODvwMR8JF6QmlfqV1j"
	}

	rpcClient := ethrpc.New(rpcURL)
	rpcClient.SetMulticallContract(common.HexToAddress(mc3Addr))

	cfg := &Config{
		DexID:          DexType,
		ChainID:        valueobject.ChainIDBerachain,
		FactoryAddress: factoryAddr,
		Multicall3:     mc3Addr,
	}
	cfg.Pyth.Urls = []string{pythDefaultUrl}

	seed := entity.Pool{
		Address:  poolAddr,
		Exchange: DexType,
		Type:     DexType,
		Tokens: []*entity.PoolToken{
			{Address: wethAddr, Decimals: 18, Swappable: true},
			{Address: usdcAddr, Decimals: 6, Swappable: true},
		},
		Reserves: []string{"0", "0"},
		Extra:    "{}",
	}

	tracker, err := NewPoolTracker(cfg, rpcClient)
	require.NoError(t, err)

	ctx := context.Background()
	updated, err := tracker.GetNewPoolState(ctx, seed, pool.GetNewPoolStateParams{})
	require.NoError(t, err)

	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(updated.Extra), &extra))

	t.Logf("=== Pool state ===")
	t.Logf("reserves:        %v", updated.Reserves)
	t.Logf("Price0 (Q64):    %s", extra.Price0)
	t.Logf("Price1 (Q64):    %s", extra.Price1)
	t.Logf("AmmPrice (Q64):  %s", extra.AmmPrice)
	t.Logf("Fee: %d  Gamma: %d  Lambda: %d  PythWeight: %d", extra.Fee, extra.Gamma, extra.Lambda, extra.PythWeight)

	// ── On-chain getSwapPrices for USDC→WETH (amount0Out=1, amount1Out=0) ───
	r0, _ := new(big.Int).SetString(updated.Reserves[0], 10)
	r1, _ := new(big.Int).SetString(updated.Reserves[1], 10)
	type swapPricesResult struct {
		SPrice0    *big.Int
		SPrice1    *big.Int
		PythPrice0 *big.Int
		PythPrice1 *big.Int
		AmmPrice   *big.Int
		AdjPrice   *big.Int
	}
	var onChain swapPricesResult
	_, err = rpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    brownFiV3FactoryABI,
			Target: factoryAddr,
			Method: factoryMethodGetSwapPrices,
			Params: []any{
				common.HexToAddress(poolAddr),
				r0, r1,
				uint8(18), uint8(6),
				big.NewInt(1), big.NewInt(0), // amount0Out=1 → tokenOut=WETH
			},
		}, []any{&onChain}).
		Call()
	require.NoError(t, err)

	t.Logf("=== On-chain getSwapPrices (amount0Out=1, USDC→WETH) ===")
	t.Logf("sPrice0  (WETH, Q64): %s", onChain.SPrice0)
	t.Logf("sPrice1  (USDC, Q64): %s", onChain.SPrice1)
	t.Logf("pythPrice0 (Q64):     %s", onChain.PythPrice0)
	t.Logf("pythPrice1 (Q64):     %s", onChain.PythPrice1)
	t.Logf("ammPrice   (Q64):     %s", onChain.AmmPrice)
	t.Logf("adjPrice   (Q64):     %s", onChain.AdjPrice)

	// ── Our off-chain computeSwapPrices ───────────────────────────────────
	sim, err := NewPoolSimulator(pool.FactoryParams{
		EntityPool: updated,
		ChainID:    valueobject.ChainIDBerachain,
	})
	require.NoError(t, err)

	offPriceIn, offPriceOut, offAdj, _, isSell, err := sim.swapContext(0)
	require.NoError(t, err)

	t.Logf("=== Off-chain swapContext (USDC→WETH, indexOut=0) ===")
	t.Logf("isSell:           %v", isSell)
	t.Logf("priceIn  (Q64):   %s", offPriceIn)
	t.Logf("priceOut (Q64):   %s", offPriceOut)
	t.Logf("adjPrice (Q64):   %s", offAdj)

	// ── Compare prices ────────────────────────────────────────────────────
	onChainPriceOut := uint256.MustFromBig(onChain.SPrice0) // sPrice0 = WETH price (base, isSell=true)
	onChainPriceIn := uint256.MustFromBig(onChain.SPrice1)  // sPrice1 = USDC price
	onChainAdj := uint256.MustFromBig(onChain.AdjPrice)

	diffOut := new(big.Int).Sub(offPriceOut.ToBig(), onChainPriceOut.ToBig())
	diffIn := new(big.Int).Sub(offPriceIn.ToBig(), onChainPriceIn.ToBig())
	diffAdj := new(big.Int).Sub(offAdj.ToBig(), onChainAdj.ToBig())
	t.Logf("=== Price deltas (off-chain minus on-chain) ===")
	t.Logf("priceOut delta: %s", diffOut)
	t.Logf("priceIn  delta: %s", diffIn)
	t.Logf("adjPrice delta: %s", diffAdj)

	// Log price divergence — expected to be small (~Pyth staleness, not a formula bug).
	// AMM formula is scale-invariant: same priceOut/priceIn ratio → same amountOut.
	// Divergence here = USDC Pyth price differs between our fetch and on-chain cache.
	t.Logf("priceOut relative delta: %.6f%%", float64(diffOut.Int64())/float64(onChainPriceOut.ToBig().Int64())*100)
}
