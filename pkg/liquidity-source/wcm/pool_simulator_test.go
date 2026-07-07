package wcm

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestSimulation(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("skip")
	}

	ctx := context.Background()
	cfg := &Config{
		DexID:           "wcm",
		ExchangeAddress: "0x5e3Ae52EbA0F9740364Bd5dd39738e1336086A8b",
		MaxOrderLevels:  maxOrderBookLevels,
	}
	rpcClient := ethrpc.New("https://mainnet.megaeth.com/rpc").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	updater := NewPoolsListUpdater(cfg, rpcClient)
	pools, _, err := updater.GetNewPools(ctx, nil)
	require.NoError(t, err)

	tracker, _ := NewPoolTracker(cfg, rpcClient)

	for _, p := range pools {
		t.Run(p.Address, func(t *testing.T) {
			newState, err := tracker.GetNewPoolState(ctx, p, pool.GetNewPoolStateParams{})
			if err != nil {
				t.Logf("  skipping: failed to get state: %v", err)
				return
			}

			for i, tok := range newState.Tokens {
				var decimals uint8
				req := rpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
					ABI:    erc20ABI,
					Target: tok.Address,
					Method: "decimals",
					Params: nil,
				}, []any{&decimals})
				if _, err := req.Call(); err == nil {
					newState.Tokens[i].Decimals = decimals
				}
			}

			p.Reserves = newState.Reserves
			p.Extra = newState.Extra
			ps, err := NewPoolSimulator(p)
			if err != nil {
				t.Logf("  skipping: failed to create simulator: %v", err)
				return
			}

			directions := []struct {
				name     string
				tokenIn  string
				tokenOut string
				isBuy    bool // In = Quote, Out = Base
			}{
				{"Sell (Base->Quote)", ps.Info.Tokens[0], ps.Info.Tokens[1], false},
				{"Buy (Quote->Base)", ps.Info.Tokens[1], ps.Info.Tokens[0], true},
			}

			for _, dir := range directions {
				var capacityBasePD *big.Int
				if dir.isBuy {
					capacityBasePD = new(big.Int)
					for _, l := range ps.Extra.OrderBook.Asks {
						capacityBasePD.Add(capacityBasePD, l.Quantity)
					}
				} else {
					capacityBasePD = new(big.Int)
					for _, l := range ps.Extra.OrderBook.Bids {
						capacityBasePD.Add(capacityBasePD, l.Quantity)
					}
				}

				if capacityBasePD.Sign() <= 0 {
					continue
				}

				targetBasePD := new(big.Int).Div(capacityBasePD, big.NewInt(10))
				if targetBasePD.Sign() <= 0 {
					targetBasePD.SetInt64(1)
				}

				var totalAmountIn *big.Int
				if dir.isBuy {
					tempSim := ps.CloneState().(*PoolSimulator)
					totalGrossQuotePD, _, err := tempSim.executeAskOrders(targetBasePD)
					if err != nil {
						continue
					}
					totalAmountIn = scaleAmountDecimals(totalGrossQuotePD, ps.StaticExtra.BuyTokenPositionDecimals, ps.payTokenDecs)
				} else {
					totalAmountIn = scaleAmountDecimals(targetBasePD, ps.StaticExtra.BuyTokenPositionDecimals, ps.buyTokenDecs)
				}

				N := new(big.Int).Div(totalAmountIn, big.NewInt(20))
				if N.Sign() <= 0 {
					continue
				}
				totalAmountIn.Mul(N, big.NewInt(20))

				sim1 := ps.CloneState().(*PoolSimulator)
				sim2 := ps.CloneState().(*PoolSimulator)

				totalAmountOut1 := new(big.Int)
				var lastErr1 error
				for i := 0; i < 20; i++ {
					res, err := sim1.CalcAmountOut(pool.CalcAmountOutParams{
						TokenAmountIn: pool.TokenAmount{Token: dir.tokenIn, Amount: N},
						TokenOut:      dir.tokenOut,
					})
					if err != nil {
						lastErr1 = err
						break
					}
					totalAmountOut1.Add(totalAmountOut1, res.TokenAmountOut.Amount)
					sim1.UpdateBalance(pool.UpdateBalanceParams{
						TokenAmountIn:  pool.TokenAmount{Token: dir.tokenIn, Amount: N},
						TokenAmountOut: *res.TokenAmountOut,
						SwapInfo:       res.SwapInfo,
					})
				}

				res2, err2 := sim2.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: dir.tokenIn, Amount: totalAmountIn},
					TokenOut:      dir.tokenOut,
				})

				if lastErr1 != nil || err2 != nil {
					continue
				}

				amountOut1 := totalAmountOut1
				amountOut2 := res2.TokenAmountOut.Amount

				diff := new(big.Int).Sub(amountOut1, amountOut2)
				diff.Abs(diff)
				bps := new(big.Int).Mul(diff, big.NewInt(10000))
				if amountOut2.Sign() > 0 {
					bps.Div(bps, amountOut2)
				}

				if bps.Int64() >= 10 {
					t.Errorf("%s: pool %s deviation too high: %d bps", dir.name, p.Address, bps.Int64())
				}
			}
		})
	}
}
