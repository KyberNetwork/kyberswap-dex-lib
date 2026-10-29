package kuruobv2

import (
	"context"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var estimateSwapABI = lo.Must(abi.JSON(strings.NewReader(`[{"type":"function","name":"estimateSwap","inputs":[{"name":"isBuy","type":"bool","internalType":"bool"},{"name":"amountIn","type":"uint128","internalType":"uint128"}],"outputs":[{"name":"result","type":"tuple","internalType":"struct ISpotOrderBook.SwapResult","components":[{"name":"amountInUsed","type":"uint128","internalType":"uint128"},{"name":"amountOut","type":"uint128","internalType":"uint128"}]}],"stateMutability":"view"}]`)))

// Runs against Monad mainnet. The sim must equal estimateSwap at the tracked block, to the wei.
func TestListAndTrack_Mainnet(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}
	rpc := ethrpc.New("https://rpc.monad.xyz").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	cfg := &Config{DexID: DexType, ChainId: valueobject.ChainIDMonad,
		AccountCore: "0x264123a9b96cc1193944cc25c188f1bd3dfa670e",
		HTTPConfig:  HTTPConfig{BaseURL: "https://api.mainnet.kuru.io"}}

	lister := NewPoolListUpdater(cfg, rpc)
	pools, metadata, err := lister.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.NotEmpty(t, pools)

	tracker := NewPoolTracker(cfg, rpc)
	for _, p := range pools {
		p.Tokens[0].Decimals, p.Tokens[1].Decimals = 18, 6 // set by pool-service in prod; unused by the sim
		p, err = tracker.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
		t.Logf("%s reserves=%v block=%d static=%s", p.Address, p.Reserves, p.BlockNumber, p.StaticExtra)

		sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
		require.NoError(t, err)
		if len(sim.Bids) == 0 || len(sim.Asks) == 0 {
			continue
		}
		// Sells: base-size units times the multiplier; buys: quote atoms (0.1 to 100k USDC).
		var amounts [2][]*big.Int
		for _, units := range []int64{1e3, 1e5, 1e7, 1e9} {
			amounts[0] = append(amounts[0], new(big.Int).Mul(sim.BaseSizeMultiplier.ToBig(), big.NewInt(units)))
		}
		for _, q := range []int64{1e5, 1e6, 1e7, 1e8, 1e9, 1e10, 1e11} {
			amounts[1] = append(amounts[1], big.NewInt(q))
		}
		for idxIn := range amounts {
			for _, amountIn := range amounts[idxIn] {
				var est struct {
					Result struct{ AmountInUsed, AmountOut *big.Int } `abi:"result"`
				}
				_, err = rpc.NewRequest().SetBlockNumber(new(big.Int).SetUint64(p.BlockNumber)).
					AddCall(&ethrpc.Call{ABI: estimateSwapABI, Target: p.Address, Method: "estimateSwap",
						Params: []any{idxIn == 1, amountIn}}, []any{&est}).Call()
				require.NoError(t, err)
				res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: p.Tokens[idxIn].Address, Amount: amountIn},
					TokenOut:      p.Tokens[1-idxIn].Address})
				if err != nil {
					t.Logf("%s idxIn=%d in=%v: sim err=%v est=%v", p.Address, idxIn, amountIn, err, est.Result)
					require.Zero(t, est.Result.AmountOut.Sign(), p.Address) // chain must agree it is unfillable
					continue
				}
				t.Logf("%s idxIn=%d in=%v out=%v", p.Address, idxIn, amountIn, res.TokenAmountOut.Amount)
				require.Equal(t, est.Result.AmountOut.String(), res.TokenAmountOut.Amount.String(), "%s in=%v", p.Address, amountIn)
				require.Equal(t, est.Result.AmountInUsed.String(),
					new(big.Int).Sub(amountIn, res.RemainingTokenAmountIn.Amount).String(), p.Address)
			}
		}
	}

	newPools, _, err := lister.GetNewPools(context.Background(), metadata)
	require.NoError(t, err)
	require.Empty(t, newPools)
}
