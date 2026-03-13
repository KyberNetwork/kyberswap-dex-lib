package kipseliprop

import (
	"context"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestKipseliDebug_QuoteVsSim(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip()
	}

	const (
		routerAddr = "0x5e4f46e92311685b590fb65128f4fe17034ac7e1"
		lensAddr   = "0x62aff80b3d2AfE0e497f1Ef735a6fDC9c3ef1acf"
		weth       = "0x4200000000000000000000000000000000000006"
		usdc       = "0x833589fCD6eDb6E08f4C7C32D4f71b54bdA02913"
	)

	cfg := Config{
		DexID:         DexType,
		ChainID:       8453,
		LensAddress:   lensAddr,
		RouterAddress: routerAddr,
		Buffer:        10000,
	}

	rpcClient := ethrpc.New("https://base.kyberengineering.io").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	inputPool := entity.Pool{
		Address: DexType + "_" + weth + "_" + usdc,
		Tokens: []*entity.PoolToken{
			{Address: weth, Decimals: 18, Swappable: true},
			{Address: usdc, Decimals: 6, Swappable: true},
		},
		Reserves:    []string{"0", "0"},
		StaticExtra: "{}",
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
				// On-chain quote
				var quoterOut *big.Int
				req := rpcClient.NewRequest().SetContext(context.Background())
				if p.BlockNumber > 0 {
					req.SetBlockNumber(new(big.Int).SetUint64(p.BlockNumber))
				}
				req.AddCall(&ethrpc.Call{
					ABI:    swapABI,
					Target: cfg.RouterAddress,
					Method: "quote",
					Params: []any{dir.tokenIn, amt, dir.tokenOut},
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
						t.Errorf("quoter reverted/zero but simulator returned positive amount: %s", simRes.TokenAmountOut.Amount.String())
					}
					return
				}

				if simErr != nil || simRes == nil {
					t.Errorf("quoter OK but simulator failed: %v", simErr)
					return
				}

				bps := calculateBPS(quoterOut, simRes.TokenAmountOut.Amount)
				if bps > 50 {
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
