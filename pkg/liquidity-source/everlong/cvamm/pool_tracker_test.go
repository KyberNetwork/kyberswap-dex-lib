package everlongcvamm

import (
	"bytes"
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Live pipeline test: lister -> tracker -> simulator against a real deployment, plus a
// wei-exact quote parity check against the on-chain CvammQuoter when one is configured.
// The venue is not deployed yet (Berachain pending), so the test is env-gated:
//
//	EVERLONG_CVAMM_RPC_URL  RPC endpoint (required)
//	EVERLONG_CVAMM_ALM      CvammALM address (required)
//	EVERLONG_CVAMM_QUOTER   CvammQuoter address (optional — enables wei-exact parity)
func TestLivePipeline(t *testing.T) {
	rpcURL := os.Getenv("EVERLONG_CVAMM_RPC_URL")
	almAddress := os.Getenv("EVERLONG_CVAMM_ALM")
	if rpcURL == "" || almAddress == "" {
		t.Skip("EVERLONG_CVAMM_RPC_URL / EVERLONG_CVAMM_ALM not set")
	}
	quoterAddress := os.Getenv("EVERLONG_CVAMM_QUOTER")

	// Multicall3 canonical address; override for chains that deploy it elsewhere.
	multicall := os.Getenv("EVERLONG_CVAMM_MULTICALL")
	if multicall == "" {
		multicall = "0xcA11bde05977b3631167028862bE2a173976CA11"
	}
	client := ethrpc.New(rpcURL).SetMulticallContract(common.HexToAddress(multicall))
	cfg := &Config{
		DexID: DexType,
		ALMs:  []ALMConfig{{Address: almAddress, Quoter: quoterAddress}},
	}
	ctx := context.Background()

	pools, _, err := NewPoolsListUpdater(cfg, client).GetNewPools(ctx, nil)
	require.NoError(t, err)
	require.Len(t, pools, 1)
	require.Len(t, pools[0].Tokens, 2)

	tracked, err := NewPoolTracker(cfg, client).GetNewPoolState(ctx, pools[0], pool.GetNewPoolStateParams{})
	require.NoError(t, err)

	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(tracked.Extra), &extra))
	require.NotNil(t, extra.XWad)
	require.NotNil(t, extra.Kappa)
	t.Logf("block=%d x=%s kappa=%s fees=(%s/%s) paused=%v reserves=%v",
		tracked.BlockNumber, extra.XWad.Dec(), extra.Kappa.Dec(),
		extra.FeeStableInWad.Dec(), extra.FeeVolatileInWad.Dec(), extra.Paused, tracked.Reserves)

	sim, err := NewPoolSimulator(tracked)
	require.NoError(t, err)

	if quoterAddress == "" {
		t.Log("EVERLONG_CVAMM_QUOTER not set — skipping wei-exact quote parity")
		return
	}
	quoterABI, err := abi.JSON(bytes.NewReader([]byte(`[{
		"inputs": [
			{"internalType": "address", "name": "alm", "type": "address"},
			{"internalType": "bool", "name": "stableIn", "type": "bool"},
			{"internalType": "uint256", "name": "amountIn", "type": "uint256"}
		],
		"name": "quoteExactInput",
		"outputs": [
			{"internalType": "uint256", "name": "amountOut", "type": "uint256"},
			{"internalType": "uint160", "name": "sqrtPriceX96After", "type": "uint160"},
			{"internalType": "uint256", "name": "unspent", "type": "uint256"}
		],
		"stateMutability": "view",
		"type": "function"
	}]`)))
	require.NoError(t, err)

	// A spread of sizes per direction, quoted at the tracked block so the states match.
	blockNumber := new(big.Int).SetUint64(tracked.BlockNumber)
	for _, stableIn := range []bool{true, false} {
		reserveIdx := 1 // volatile reserve bounds stable-in output
		if !stableIn {
			reserveIdx = 0
		}
		outReserve, ok := new(big.Int).SetString(tracked.Reserves[reserveIdx], 10)
		require.True(t, ok)
		if outReserve.Sign() == 0 {
			continue
		}
		for _, div := range []int64{1000000, 1000, 10, 1} {
			amountIn := new(big.Int).Quo(outReserve, big.NewInt(div))
			if amountIn.Sign() == 0 {
				continue
			}
			var quoted struct {
				AmountOut         *big.Int
				SqrtPriceX96After *big.Int
				Unspent           *big.Int
			}
			req := client.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)
			req.AddCall(&ethrpc.Call{
				ABI:    quoterABI,
				Target: quoterAddress,
				Method: "quoteExactInput",
				Params: []any{common.HexToAddress(almAddress), stableIn, amountIn},
			}, []any{&quoted})
			_, err := req.Aggregate()
			require.NoError(t, err)

			tokenIn, tokenOut := sim.Info.Tokens[0], sim.Info.Tokens[1]
			if !stableIn {
				tokenIn, tokenOut = tokenOut, tokenIn
			}
			res, calcErr := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amountIn},
				TokenOut:      tokenOut,
			})
			if quoted.AmountOut == nil || quoted.AmountOut.Sign() == 0 {
				require.Error(t, calcErr, "stableIn=%v div=%d: chain quotes zero, sim must reject", stableIn, div)
				continue
			}
			require.NoError(t, calcErr, "stableIn=%v div=%d", stableIn, div)
			require.Equal(t, quoted.AmountOut.String(), res.TokenAmountOut.Amount.String(),
				"stableIn=%v div=%d amountOut", stableIn, div)
			require.Equal(t, quoted.Unspent.String(), res.RemainingTokenAmountIn.Amount.String(),
				"stableIn=%v div=%d unspent", stableIn, div)
		}
	}
}
