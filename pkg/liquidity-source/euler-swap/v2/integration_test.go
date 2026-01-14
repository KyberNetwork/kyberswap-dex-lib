package v2

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/euler-swap/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_V2_ComputeQuote(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	rpcClient := ethrpc.New("https://ethereum.kyberengineering.io")
	rpcClient.SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	registryAddress := "0x5fccb84363f020c0cade052c9c654aabf932814a"

	config := shared.Config{
		FactoryAddress: registryAddress,
	}

	updater := NewPoolsListUpdater(&config, rpcClient)
	tracker, err := NewPoolTracker(&config, rpcClient)
	require.NoError(t, err)

	ctx := context.Background()
	metadata := []byte(`{"offset": 0}`)
	pools, _, err := updater.GetNewPools(ctx, metadata)
	require.NoError(t, err)

	if len(pools) == 0 {
		t.Skip("No pools found")
	}

	for _, p := range pools {
		t.Run(p.Address, func(t *testing.T) {
			updatedPool, err := tracker.GetNewPoolState(ctx, p, pool.GetNewPoolStateParams{})
			require.NoError(t, err)

			simulator, err := NewPoolSimulator(updatedPool)
			require.NoError(t, err)

			tokenIn := updatedPool.Tokens[0].Address
			tokenOut := updatedPool.Tokens[1].Address

			amounts := []*big.Int{
				bignumber.TenPowInt(18),
				bignumber.TenPowInt(15),
				bignumber.TenPowInt(12),
				bignumber.TenPowInt(10),
				bignumber.TenPowInt(6),
			}

			success := false
			for _, amountIn := range amounts {
				res, err := simulator.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{
						Token:  tokenIn,
						Amount: amountIn,
					},
					TokenOut: tokenOut,
				})
				if err != nil {
					continue
				}
				success = true

				// On-chain computeQuote
				var amountOutOnChain *big.Int
				req := rpcClient.NewRequest().SetContext(ctx)
				req.AddCall(&ethrpc.Call{
					ABI:    poolABI,
					Target: updatedPool.Address,
					Method: "computeQuote",
					Params: []any{
						common.HexToAddress(tokenIn),
						common.HexToAddress(tokenOut),
						amountIn,
						true,
					},
				}, []any{&amountOutOnChain})

				_, err = req.Call()
				require.NoError(t, err)

				fmt.Printf("Pool: %s, In: %s, Out Simulator: %s, Out On-chain: %s\n",
					updatedPool.Address, amountIn.String(), res.TokenAmountOut.Amount.String(), amountOutOnChain.String())

				diff := new(big.Int).Abs(new(big.Int).Sub(res.TokenAmountOut.Amount, amountOutOnChain))
				bps := new(big.Int).Div(new(big.Int).Mul(diff, big.NewInt(10000)), amountOutOnChain)

				assert.Less(t, bps.Int64(), int64(10), "Difference should be less than 10 BPS")
				break
			}
			assert.True(t, success, "Should at least succeed for one amount")
		})
	}
}
