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

// Runs against Monad testnet: V2 has no mainnet deployment yet.
func TestListAndTrack_Testnet(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}
	rpc := ethrpc.New("https://testnet-rpc.monad.xyz").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	cfg := &Config{DexID: DexType, ChainId: valueobject.ChainIDMonad,
		AccountCore: "0xdbaade7b42c95399bb1e9614d51b5b9e2cf78038",
		HTTPConfig:  HTTPConfig{BaseURL: "https://api.testnet.kuru.io"}}

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

		sim, err := NewPoolSimulator(p)
		require.NoError(t, err)
		// Quote 1 USDC buy and a 1e5-book-unit sell; both must equal estimateSwap at the tracked block.
		for idxIn, amountIn := range []*big.Int{new(big.Int).Mul(sim.BaseSizeMultiplier.ToBig(), big.NewInt(1e5)),
			big.NewInt(1e6)} {
			if len(sim.Bids) == 0 || len(sim.Asks) == 0 {
				break
			}
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
			require.NoError(t, err)
			require.Equal(t, est.Result.AmountOut.String(), res.TokenAmountOut.Amount.String(), p.Address)
			require.Equal(t, est.Result.AmountInUsed.String(),
				new(big.Int).Sub(amountIn, res.RemainingTokenAmountIn.Amount).String(), p.Address)
		}
	}

	newPools, _, err := lister.GetNewPools(context.Background(), metadata)
	require.NoError(t, err)
	require.Empty(t, newPools)
}
