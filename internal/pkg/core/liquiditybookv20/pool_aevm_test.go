package liquiditybookv20

import (
	"fmt"
	"math/big"
	"testing"

	aevmclient "github.com/KyberNetwork/aevm/client"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	routerentity "github.com/KyberNetwork/router-service/internal/pkg/entity"
)

const (
	aevmServerURL = "localhost:8246" // CHANGE THIS

	usdceUSDCPool = "0x18332988456c4bd9aba6698ec748b331516f5a14"

	usdceAddr = "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664"
	usdcAddr  = "0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e"
)

var (
	balanceSlots = map[common.Address]*routerentity.ERC20BalanceSlot{
		common.HexToAddress(usdceAddr): {
			Token:       usdceAddr,
			Wallet:      "0x47f3c2557364efc28f1269e3169773fa5236384d",
			BalanceSlot: "0x4f1749155d837e5f5ef076382254c01af904c6ddb97b100fef402248f448ea99",
		},
		common.HexToAddress(usdcAddr): {
			Token:       usdcAddr,
			Wallet:      "0x47F3C2557364EFC28f1269e3169773fa5236384D",
			BalanceSlot: "0xcdd82b6bead1cac3d1e09d54b01220a76c9534fbd5cfb487b133d7568fced94a",
		},
	}
	balanceSlotsWithHoldersList = map[common.Address]*routerentity.ERC20BalanceSlot{
		common.HexToAddress(usdceAddr): {
			Token:  usdceAddr,
			Wallet: "0x47f3c2557364efc28f1269e3169773fa5236384d",
			Holders: []string{
				"0x3a2434c698f8d79af1f5a9e43013157ca8b11a66",
			},
		},
		common.HexToAddress(usdcAddr): {
			Token:  usdcAddr,
			Wallet: "0x47F3C2557364EFC28f1269e3169773fa5236384D",
			Holders: []string{
				"0xe86a549cebab14ddff9741fd46e62a60ebff5b23",
			},
		},
	}
)

func TestCalcAmountOutAEVMWithUSDCE_USDCPoolWithGRPCClient(t *testing.T) {
	t.Skip()

	client, err := aevmclient.NewGRPCClient(aevmServerURL)
	require.NoError(t, err)

	stateRoot, err := client.LatestStateRoot()
	fmt.Printf("stateRoot = %s\n", stateRoot)
	require.NoError(t, err)

	names := []string{"without holders lists", "with holders lists"}
	for i, balanceSlots := range []map[common.Address]*routerentity.ERC20BalanceSlot{balanceSlots, balanceSlotsWithHoldersList} {
		balanceSlots := balanceSlots
		t.Run(names[i], func(t *testing.T) {
			p, err := NewPoolAEVM(
				valueobject.ChainIDAvalancheCChain,
				entity.Pool{
					Address: usdceUSDCPool,
					Tokens: []*entity.PoolToken{
						{Address: usdceAddr},
						{Address: usdcAddr},
					},
					Reserves: entity.PoolReserves{"0", "0"},
				},
				client,
				common.Hash(stateRoot),
				balanceSlots,
			)
			require.NoError(t, err)
			result, err := p.CalcAmountOutAEVM(
				pool.TokenAmount{
					Token:  usdceAddr,
					Amount: big.NewInt(500_000_000), // 500 USDC.e
				},
				usdcAddr,
			)
			require.NoError(t, err)
			fmt.Printf("swapping 500 USDC.e for USDC amountOut = %s, gas used = %v\n", result.TokenAmountOut.Amount, result.Gas)
			usdcOut := new(big.Int).Set(result.TokenAmountOut.Amount)

			p.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: result.SwapInfo})

			result, err = p.CalcAmountOutAEVM(
				pool.TokenAmount{
					Token:  usdcAddr,
					Amount: usdcOut,
				},
				usdceAddr,
			)
			require.NoError(t, err)
			fmt.Printf("swapping %s USDC for USDC.e amountOut = %s, gas used = %v\n", usdcOut, result.TokenAmountOut.Amount, result.Gas)
		})
	}
}
