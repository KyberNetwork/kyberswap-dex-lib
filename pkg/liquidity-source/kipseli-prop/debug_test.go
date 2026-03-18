package kipseliprop

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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const (
	testRouterAddr = "0x5e4f46e92311685b590fb65128f4fe17034ac7e1"
	testLensAddr   = "0x62aff80b3d2AfE0e497f1Ef735a6fDC9c3ef1acf"
	testWeth       = "0x4200000000000000000000000000000000000006"
	testUsdc       = "0x833589fCD6eDb6E08f4C7C32D4f71b54bdA02913"
)

func setupKipseliTest(t *testing.T) (entity.Pool, *PoolSimulator, *PoolTracker, *ethrpc.Client) {
	t.Helper()
	if os.Getenv("CI") != "" {
		t.Skip()
	}

	rpcURL := os.Getenv("BASE_RPC_URL")
	if rpcURL == "" {
		rpcURL = "https://base.kyberengineering.io"
	}

	var verifier common.Address
	if v := os.Getenv("KIPSELI_VERIFIER"); v != "" {
		verifier = common.HexToAddress("")
	}

	var quoter common.Hash
	if q := os.Getenv("KIPSELI_QUOTER"); q != "" {
		quoter = common.HexToHash(q)
	}

	cfg := Config{
		DexID:         DexType,
		ChainID:       8453,
		LensAddress:   testLensAddr,
		RouterAddress: testRouterAddr,
		Verifier:      verifier,
		Quoter:        quoter,
		Buffer:        10000,
	}

	rpcClient := ethrpc.New(rpcURL).
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	inputPool := entity.Pool{
		Address: DexType + "_" + testWeth + "_" + testUsdc,
		Tokens: []*entity.PoolToken{
			{Address: testWeth, Decimals: 18, Swappable: true},
			{Address: testUsdc, Decimals: 6, Swappable: true},
		},
		Reserves:    []string{"0", "0"},
		StaticExtra: `{"routerAddress":"` + testRouterAddr + `"}`,
	}

	tracker := NewPoolTracker(&cfg, rpcClient)
	p, err := tracker.GetNewPoolState(context.Background(), inputPool, pool.GetNewPoolStateParams{})
	require.NoError(t, err)

	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)

	return p, sim, tracker, rpcClient
}

func TestKipseliDebug_QuoteVsSim(t *testing.T) {
	p, sim, tracker, rpcClient := setupKipseliTest(t)

	type direction struct {
		label    string
		tokenIn  common.Address
		tokenOut common.Address
	}
	directions := []direction{
		{"WETH=>USDC", common.HexToAddress(testWeth), common.HexToAddress(testUsdc)},
		{"USDC=>WETH", common.HexToAddress(testUsdc), common.HexToAddress(testWeth)},
	}

	tsMs := big.NewInt(time.Now().UnixMilli())

	src := rand.New(rand.NewSource(time.Now().Unix()))
	amounts := make([]*big.Int, 0, 9)
	for _, exp := range []int{6, 12, 18} {
		for i := 0; i < 3; i++ {
			n := src.Int63n(9_000_000) + 1_000_000
			amounts = append(amounts, new(big.Int).Mul(big.NewInt(n), bignumber.TenPowInt(exp-6)))
		}
	}

	for _, dir := range directions {
		sig := tracker.signQuote(dir.tokenIn, dir.tokenOut, tsMs)
		for _, amt := range amounts {
			t.Run(fmt.Sprintf("%s_%s", dir.label, amt.String()), func(t *testing.T) {
				var quoterOut *big.Int
				req := rpcClient.NewRequest().SetContext(context.Background())
				if p.BlockNumber > 0 {
					req.SetBlockNumber(new(big.Int).SetUint64(p.BlockNumber))
				}
				req.AddCall(&ethrpc.Call{
					ABI:    swapABI,
					Target: testRouterAddr,
					Method: "quote",
					Params: []any{dir.tokenIn, amt, dir.tokenOut, tsMs, sig},
				}, []any{&quoterOut})
				_, qErr := req.Call()

				simRes, simErr := sim.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: dir.tokenIn.Hex(), Amount: amt},
					TokenOut:      dir.tokenOut.Hex(),
				})

				if qErr != nil || quoterOut == nil || quoterOut.Sign() == 0 {
					if simErr == nil && simRes != nil && simRes.TokenAmountOut.Amount.Sign() > 0 {
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

func TestKipseliMergeSwap(t *testing.T) {
	p, sim1, _, _ := setupKipseliTest(t)

	sim2 := sim1.CloneState()
	require.NotNil(t, sim2)

	tokenIn := strings.ToLower(p.Tokens[0].Address)
	tokenOut := strings.ToLower(p.Tokens[1].Address)

	amountPerSwap := new(big.Int).Mul(big.NewInt(5), bignumber.TenPowInt(14)) // 0.0005 WETH
	numSwaps := 20
	totalAmount := new(big.Int).Mul(amountPerSwap, big.NewInt(int64(numSwaps)))

	limit1 := swaplimit.NewInventory(DexType, sim1.CalculateLimit())
	totalOut1 := new(big.Int)
	var err1 error
	for i := 0; i < numSwaps; i++ {
		res, err := sim1.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amountPerSwap},
			TokenOut:      tokenOut,
			Limit:         limit1,
		})
		if err != nil {
			err1 = err
			t.Logf("N-swap: failed at swap %d: %v", i+1, err)
			break
		}
		totalOut1.Add(totalOut1, res.TokenAmountOut.Amount)
		sim1.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn:  pool.TokenAmount{Token: tokenIn, Amount: amountPerSwap},
			TokenAmountOut: *res.TokenAmountOut,
			SwapLimit:      limit1,
		})
	}

	sim2Typed := sim2.(*PoolSimulator)
	res2, err2 := sim2Typed.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: totalAmount},
		TokenOut:      tokenOut,
		Limit:         swaplimit.NewInventory(DexType, sim2Typed.CalculateLimit()),
	})

	if err2 != nil {
		if err1 == nil {
			t.Errorf("single swap N*X failed (%v) but N swaps never errored (totalOut=%s)", err2, totalOut1)
			return
		}
		require.Equal(t, err1.Error(), err2.Error(), "N swaps and 1 swap should fail with the same error")
		t.Logf("both N-swap (stopped at totalOut=%s, err=%v) and 1-swap N*X hit cap: %v", totalOut1, err1, err2)
		return
	}

	if totalOut1.Sign() == 0 {
		t.Errorf("N swaps produced zero output but single swap N*X succeeded with out=%s", res2.TokenAmountOut.Amount)
		return
	}

	bps := calculateBPS(totalOut1, res2.TokenAmountOut.Amount)
	t.Logf("N swaps totalOut=%s, 1 swap totalOut=%s, BPS diff=%d", totalOut1, res2.TokenAmountOut.Amount, bps)
	require.LessOrEqual(t, bps, int64(1), "N consecutive swaps (amount X) should match 1 swap (amount N*X) within 1 BPS")
}

func calculateBPS(quoter, sim *big.Int) int64 {
	if quoter.Sign() == 0 {
		return 0
	}
	diff := new(big.Int).Abs(new(big.Int).Sub(quoter, sim))
	return new(big.Int).Div(new(big.Int).Mul(diff, bignumber.BasisPoint), quoter).Int64()
}
