package tessera

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// TestTesseraDebugFailingCases tests specific failing cases
func TestTesseraDebugFailingCases(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip()
	}

	cfg := Config{
		DexId:          "tessera",
		TesseraIndexer: "0x505352DA2918C6a06f12F3d59FFb79905d43439f",
		TesseraEngine:  "0x31E99E05fEE3DCe580aF777c3fd63Ee1b3b40c17",
		TesseraSwap:    "0x55555522005BcAE1c2424D474BfD5ed477749E3e",
	}

	rpcClient := ethrpc.New("https://base.kyberengineering.io").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	failingCases := []struct {
		poolAddr    string
		token0      string
		token1      string
		dec0        uint8
		dec1        uint8
		direction   string
		amount      *big.Int
		description string
	}{
		{
			poolAddr:    "0xed57bacdc2a990b631f8817853935791c122c356",
			token0:      "0xcbB7C0000aB88B473b1f5aFd9ef808440eed33Bf",
			token1:      "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
			dec0:        8,
			dec1:        6,
			direction:   "1=>0",
			amount:      big.NewInt(100000000), // 100 USDC
			description: "cbBTC/USDC: 100 USDC",
		},
		{
			poolAddr:    "0xe1191102bdcea1928a93b4d6ea7bf5c4e9207210",
			token0:      "0x0b3e328455c4059EEB9e3f84b5543F74E24e7E1b",
			token1:      "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
			dec0:        18,
			dec1:        6,
			direction:   "1=>0",
			amount:      big.NewInt(1000000000), // 1,000 USDC
			description: "AERO/USDC: 1000 USDC",
		},
		{
			poolAddr:    "0x3b84be4d48888a6bc385eea93e522246b214069e",
			token0:      "0x940181a94A35A4569E4529A3CDFb74e38FD98631",
			token1:      "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
			dec0:        18,
			dec1:        6,
			direction:   "1=>0",
			amount:      big.NewInt(500000000), // 500 USDC
			description: "Token/USDC: 500 USDC",
		},
	}

	for i, testCase := range failingCases {
		t.Run(fmt.Sprintf("Case_%d_%s", i, testCase.description), func(t *testing.T) {
			inputPool := entity.Pool{
				Address: testCase.poolAddr,
				Tokens: []*entity.PoolToken{
					{Address: testCase.token0, Decimals: testCase.dec0},
					{Address: testCase.token1, Decimals: testCase.dec1},
				},
				Reserves: []string{"0", "0"},
			}

			tracker := NewPoolTracker(&cfg, rpcClient)
			p, err := tracker.GetNewPoolState(context.Background(), inputPool, pool.GetNewPoolStateParams{})
			require.NoError(t, err)

			sim, err := NewPoolSimulator(p)
			require.NoError(t, err)

			var extra Extra
			err = json.Unmarshal([]byte(p.Extra), &extra)
			require.NoError(t, err)

			fmt.Printf("\n=== %s ===\n", testCase.description)
			fmt.Printf("Pool: %s\n", p.Address)
			fmt.Printf("Direction: %s, Amount: %s\n", testCase.direction, testCase.amount.String())
			fmt.Printf("TradingEnabled: %v, IsInitialised: %v\n", extra.TradingEnabled, extra.IsInitialised)
			fmt.Printf("BlockNumber: %d\n", p.BlockNumber)
			fmt.Printf("BaseToQuotePrefetches: %d items\n", len(extra.BaseToQuotePrefetches))
			fmt.Printf("QuoteToBasePrefetches: %d items\n", len(extra.QuoteToBasePrefetches))

			var tokenIn, tokenOut string
			if testCase.direction == "0=>1" {
				tokenIn = p.Tokens[0].Address
				tokenOut = p.Tokens[1].Address
			} else {
				tokenIn = p.Tokens[1].Address
				tokenOut = p.Tokens[0].Address
			}

			var quoterRes struct {
				AmountIn  *big.Int
				AmountOut *big.Int
			}
			reqQuoter := rpcClient.NewRequest().
				SetContext(context.Background()).
				SetBlockNumber(big.NewInt(int64(p.BlockNumber)))

			reqQuoter.AddCall(&ethrpc.Call{
				ABI:    TesseraRouterABI,
				Target: cfg.TesseraSwap,
				Method: "tesseraSwapViewAmounts",
				Params: []any{common.HexToAddress(tokenIn), common.HexToAddress(tokenOut), testCase.amount},
			}, []any{&quoterRes})

			_, quoterErr := reqQuoter.Call()

			if quoterErr != nil {
				fmt.Printf("Quoter REVERTED: %v\n", quoterErr)
			} else {
				fmt.Printf("Quoter output: %s\n", quoterRes.AmountOut.String())
			}

			simRes, simErr := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  tokenIn,
					Amount: testCase.amount,
				},
				TokenOut: tokenOut,
			})

			if simErr != nil {
				fmt.Printf("Simulator ERROR: %v\n", simErr)
			} else {
				fmt.Printf("Simulator output: %s\n", simRes.TokenAmountOut.Amount.String())
			}

			if quoterErr == nil && simErr == nil {
				diff := new(big.Int).Abs(new(big.Int).Sub(quoterRes.AmountOut, simRes.TokenAmountOut.Amount))
				bps := int64(0)
				if quoterRes.AmountOut.Cmp(big.NewInt(0)) > 0 {
					bps = new(big.Int).Div(new(big.Int).Mul(diff, big.NewInt(10000)), quoterRes.AmountOut).Int64()
				}
				fmt.Printf("Difference: %s, BPS: %d\n", diff.String(), bps)
				if bps > 10 {
					t.Errorf("High BPS difference: %d (expected <= 10)", bps)
				}
			} else if quoterErr != nil && simErr == nil {
				t.Errorf("Quoter reverted but simulator succeeded")
			} else if quoterErr == nil && simErr != nil {
				t.Errorf("Quoter succeeded but simulator failed")
			}
		})
	}
}
