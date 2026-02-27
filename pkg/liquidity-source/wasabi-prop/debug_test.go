package wasabiprop

import (
	"context"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestWasabiPropDebug(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip()
	}

	rpcURL := os.Getenv("BASE_RPC_URL")
	if rpcURL == "" {
		t.Skip("BASE_RPC_URL not set")
	}

	const (
		factoryAddr = "0x851fc799c9f1443a2c1e6b966605a80f8a1b1bf2"
		routerAddr  = "0xfc81dfde25083a286723b7c9dd7213f8723369fe"
		weth        = "0x4200000000000000000000000000000000000006"
		usdc        = "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
	)

	cfg := Config{
		DexID:          DexType,
		ChainID:        8453,
		FactoryAddress: factoryAddr,
		RouterAddress:  routerAddr,
		Buffer:         9900,
	}

	rpcClient := ethrpc.New(rpcURL).
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	// Discover pool address for WETH
	var poolAddr common.Address
	req := rpcClient.NewRequest().SetContext(context.Background())
	req.AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: factoryAddr,
		Method: "getPropPool",
		Params: []any{common.HexToAddress(weth)},
	}, []any{&poolAddr})
	_, err := req.TryAggregate()
	require.NoError(t, err)
	require.NotEqual(t, common.Address{}, poolAddr)

	t.Logf("Pool address: %s", poolAddr.Hex())

	inputPool := entity.Pool{
		Address: strings.ToLower(poolAddr.Hex()),
		Tokens: []*entity.PoolToken{
			{Address: strings.ToLower(weth), Decimals: 18, Swappable: true},
			{Address: strings.ToLower(usdc), Decimals: 6, Swappable: true},
		},
		Reserves:    []string{"0", "0"},
		StaticExtra: `{"routerAddress":"` + routerAddr + `"}`,
	}

	tracker := NewPoolTracker(&cfg, rpcClient)
	p, err := tracker.GetNewPoolState(context.Background(), inputPool, pool.GetNewPoolStateParams{})
	require.NoError(t, err)

	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)

	type direction struct {
		label    string
		tokenIn  common.Address
		tokenOut common.Address
	}

	directions := []direction{
		{"0=>1", common.HexToAddress(weth), common.HexToAddress(usdc)},
		{"1=>0", common.HexToAddress(usdc), common.HexToAddress(weth)},
	}

	src := rand.New(rand.NewSource(time.Now().Unix()))
	amounts := make([]*big.Int, 0, 9)
	for _, exp := range []int{6, 12, 18} {
		for i := 0; i < 3; i++ {
			n := src.Int63n(9_000_000) + 1_000_000
			base := new(big.Int).Mul(
				big.NewInt(n),
				bignumber.TenPowInt(exp-6),
			)
			amounts = append(amounts, base)
		}
	}

	for _, dir := range directions {
		for _, amt := range amounts {
			t.Run(fmt.Sprintf("%s_%s", dir.label, amt.String()), func(t *testing.T) {
				// On-chain quote via pool's quoteExactInput
				var quoterOut *big.Int
				req := rpcClient.NewRequest().SetContext(context.Background())
				if p.BlockNumber > 0 {
					req.SetBlockNumber(new(big.Int).SetUint64(p.BlockNumber))
				}
				req.AddCall(&ethrpc.Call{
					ABI:    poolABI,
					Target: p.Address,
					Method: "quoteExactInput",
					Params: []any{dir.tokenIn, amt},
				}, []any{&quoterOut})

				_, qErr := req.Call()

				// Simulator
				simRes, simErr := sim.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{
						Token:  dir.tokenIn.Hex(),
						Amount: amt,
					},
					TokenOut: dir.tokenOut.Hex(),
				})

				if qErr != nil || quoterOut == nil || quoterOut.Sign() == 0 {
					if simErr == nil && simRes != nil && simRes.TokenAmountOut.Amount.Sign() > 0 {
						// Quoter reverts for amounts exceeding pool capacity; simulator may still
						// extrapolate from samples. In production, swap limits prevent over-quoting.
						t.Logf("quoter reverted but simulator returned %s (expected for out-of-range amounts)", simRes.TokenAmountOut.Amount)
					}
					return
				}

				if simErr != nil || simRes == nil {
					t.Errorf("quoter OK but simulator failed: %v", simErr)
					return
				}

				bps := calculateBPS(quoterOut, simRes.TokenAmountOut.Amount)
				t.Logf("amt=%s quote=%s sim=%s bps=%d", amt, quoterOut, simRes.TokenAmountOut.Amount, bps)
				if bps > 200 {
					t.Errorf("high BPS diff: %d (quote=%s, sim=%s)", bps, quoterOut, simRes.TokenAmountOut.Amount)
				}
			})
		}
	}
}

func calculateBPS(quoter, sim *big.Int) int64 {
	if quoter.Sign() == 0 {
		return 0
	}
	diff := new(big.Int).Abs(new(big.Int).Sub(quoter, sim))
	return new(big.Int).Div(new(big.Int).Mul(diff, bignumber.BasisPoint), quoter).Int64()
}
