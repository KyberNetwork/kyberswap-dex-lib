package tessera

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestTesseraIntegration(t *testing.T) {
	cfg := Config{
		DexId:          "tessera",
		TesseraIndexer: "0x505352DA2918C6a06f12F3d59FFb79905d43439f",
		TesseraEngine:  "0x31E99E05fEE3DCe580aF777c3fd63Ee1b3b40c17",
		TesseraSwap:    "0x55555522005BcAE1c2424D474BfD5ed477749E3e",
	}

	rpcClient := ethrpc.New("https://base.kyberengineering.io").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	updater := NewPoolsListUpdater(&cfg, rpcClient)
	pools, _, err := updater.GetNewPools(context.Background(), nil)
	require.NoError(t, err)

	if len(pools) == 0 {
		t.Skip("No pools found")
	}

	var passed, failed int

	for i, inputPool := range pools {
		t.Run(fmt.Sprintf("Pool_%d", i), func(t *testing.T) {
			if len(inputPool.Tokens) < 2 {
				t.Skip("Insufficient tokens")
			}

			tracker := NewPoolTracker(&cfg, rpcClient)
			p, err := tracker.GetNewPoolState(context.Background(), inputPool, pool.GetNewPoolStateParams{})
			require.NoError(t, err)

			var extra Extra
			_ = json.Unmarshal([]byte(p.Extra), &extra)

			res0, _ := new(big.Int).SetString(p.Reserves[0], 10)
			res1, _ := new(big.Int).SetString(p.Reserves[1], 10)

			directions := []struct {
				label    string
				tokenIn  common.Address
				tokenOut common.Address
				max      *big.Int
			}{
				{"0=>1", common.HexToAddress(p.Tokens[0].Address), common.HexToAddress(p.Tokens[1].Address), res0},
				{"1=>0", common.HexToAddress(p.Tokens[1].Address), common.HexToAddress(p.Tokens[0].Address), res1},
			}

			for _, dir := range directions {
				amounts := generateTestAmounts(dir.max)

				for _, amt := range amounts {
					quoterOut := callQuoter(rpcClient, cfg.TesseraSwap, dir.tokenIn, dir.tokenOut, amt, p.BlockNumber)

					sim, _ := NewPoolSimulator(p)
					simRes, simErr := sim.CalcAmountOut(pool.CalcAmountOutParams{
						TokenAmountIn: pool.TokenAmount{Token: dir.tokenIn.Hex(), Amount: amt},
						TokenOut:      dir.tokenOut.Hex(),
					})

					if quoterOut == nil || quoterOut.Sign() == 0 {
						if simErr != nil {
							passed++
						}
						continue
					}

					if simErr != nil {
						if amt.Cmp(dir.max) > 0 {
							passed++ // OOR case
						} else {
							t.Errorf("%s Amt=%s: Quoter OK but Sim failed", dir.label, amt)
							failed++
						}
						continue
					}

					bps := calculateBPS(quoterOut, simRes.TokenAmountOut.Amount)
					if bps > 10 {
						t.Errorf("%s Amt=%s BPS=%d", dir.label, amt, bps)
						failed++
					} else {
						passed++
					}
				}
			}
		})
	}

	fmt.Printf("\n--- SUMMARY: Passed=%d Failed=%d ---\n", passed, failed)
}

// generateTestAmounts creates test amounts using:
// - Logarithmic scale (10^3 to 10^30) within orderbook capacity
// - Over-limit cases (2x and 10x max) to verify revert behavior
func generateTestAmounts(max *big.Int) []*big.Int {
	amounts := make([]*big.Int, 0)

	for exp := uint8(3); exp <= 30; exp += 3 {
		amt := big256.TenPow(exp).ToBig()
		if max == nil || amt.Cmp(max) <= 0 {
			amounts = append(amounts, amt)
		}
	}

	if max != nil && max.Sign() > 0 {
		amounts = append(amounts, new(big.Int).Mul(max, bignumber.Two))
		amounts = append(amounts, new(big.Int).Mul(max, bignumber.Ten))
	}

	return amounts
}

func callQuoter(client *ethrpc.Client, router string, tokenIn, tokenOut common.Address, amt *big.Int, block uint64) *big.Int {
	var res struct {
		AmountIn  *big.Int
		AmountOut *big.Int
	}

	req := client.NewRequest().SetContext(context.Background())
	if block > 0 {
		req.SetBlockNumber(new(big.Int).SetUint64(block))
	}

	req.AddCall(&ethrpc.Call{
		ABI:    TesseraRouterABI,
		Target: router,
		Method: "tesseraSwapViewAmounts",
		Params: []any{tokenIn, tokenOut, amt},
	}, []any{&res})

	if _, err := req.Call(); err != nil {
		return nil
	}

	return res.AmountOut
}

func calculateBPS(quoter, sim *big.Int) int64 {
	if quoter.Sign() == 0 {
		return 0
	}
	diff := new(big.Int).Abs(new(big.Int).Sub(quoter, sim))
	return new(big.Int).Div(new(big.Int).Mul(diff, bignumber.BasisPoint), quoter).Int64()
}
