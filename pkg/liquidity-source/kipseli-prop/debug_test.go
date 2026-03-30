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
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const (
	testRouterAddr = "0x71C2Ed90CC288229Be59F26b8B3EEF3C07d7ab99"
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
		rpcURL = "https://base-rpc.kyberswap.com"
	}

	verifier := common.HexToAddress(os.Getenv("VERIFIER_ADDRESS"))
	quoter := common.HexToHash(os.Getenv("QUOTER_HASH"))

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

func signQuoteForTest(tracker *PoolTracker, tokenIn, tokenOut common.Address, tsMs *big.Int) []byte {
	typedMsg := DomainType
	typedMsg.Domain.ChainId = math.NewHexOrDecimal256(int64(tracker.cfg.ChainID))
	typedMsg.Domain.VerifyingContract = hexutil.Encode(tracker.cfg.Verifier[:])
	typedMsg.Message = apitypes.TypedDataMessage{
		"tokenIn":            [20]byte(tokenIn),
		"tokenOut":           [20]byte(tokenOut),
		"timestampInMilisec": tsMs,
	}
	sig, _ := tracker.signer.Sign(typedMsg)
	return sig
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

	src := rand.New(rand.NewSource(time.Now().Unix()))
	amounts := make([]*big.Int, 0, 9)
	for _, exp := range []int{6, 12, 18} {
		for range 3 {
			n := src.Int63n(9_000_000) + 1_000_000
			amounts = append(amounts, new(big.Int).Mul(big.NewInt(n), bignumber.TenPowInt(exp-6)))
		}
	}

	limit := swaplimit.NewInventory(DexType, sim.CalculateLimit())

	for _, dir := range directions {
		for _, amt := range amounts {
			t.Run(fmt.Sprintf("%s_%s", dir.label, amt.String()), func(t *testing.T) {
				tsMs := big.NewInt(time.Now().UnixMilli())
				sig := signQuoteForTest(tracker, dir.tokenIn, dir.tokenOut, tsMs)

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
					Limit:         limit,
				})

				if qErr != nil || quoterOut == nil || quoterOut.Sign() == 0 {
					if simErr == nil && simRes != nil && simRes.TokenAmountOut.Amount.Sign() > 0 {
						t.Errorf("quoter reverted but simulator accepted (out=%s) — simulator overestimates", simRes.TokenAmountOut.Amount)
					}
					return
				}

				if simErr != nil || simRes == nil {
					outIdx := sim.GetTokenIndex(strings.ToLower(dir.tokenOut.Hex()))
					if outIdx >= 0 && quoterOut.Cmp(sim.GetReserves()[outIdx]) >= 0 {
						t.Logf("quoter %s >= reserve %s → exceeds inventory", quoterOut, sim.GetReserves()[outIdx])
						return
					}
					t.Errorf("quoter OK (out=%s) but simulator failed: %v", quoterOut, simErr)
					return
				}

				bps := calculateBPS(quoterOut, simRes.TokenAmountOut.Amount)
				t.Logf("amt=%s quote=%s sim=%s bps=%d", amt, quoterOut, simRes.TokenAmountOut.Amount, bps)
				if bps > 50 {
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
	for i := range numSwaps {
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
