package liquiditybookv21

import (
	"context"
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

	btcbUSDCPool = "0x4224f6f4c9280509724db2dbac314621e4465c29"

	btcbAddr = "0x152b9d0fdc40c096757f570a51e494bd4b943e50"
	usdcAddr = "0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e"
)

var (
	balanceSlots = map[common.Address]*routerentity.ERC20BalanceSlot{
		common.HexToAddress(btcbAddr): {
			Token:       btcbAddr,
			Wallet:      "0x47F3C2557364EFC28f1269e3169773fa5236384D",
			BalanceSlot: "0x4f1749155d837e5f5ef076382254c01af904c6ddb97b100fef402248f448ea99",
		},
		common.HexToAddress(usdcAddr): {
			Token:       usdcAddr,
			Wallet:      "0x47F3C2557364EFC28f1269e3169773fa5236384D",
			BalanceSlot: "0xcdd82b6bead1cac3d1e09d54b01220a76c9534fbd5cfb487b133d7568fced94a",
		},
	}
	balanceSlotsWithHoldersList = map[common.Address]*routerentity.ERC20BalanceSlot{
		common.HexToAddress(btcbAddr): {
			Token:  btcbAddr,
			Wallet: "0x47F3C2557364EFC28f1269e3169773fa5236384D",
			Holders: []string{
				"0x0000000000000000000000000000000000000000",
				"0x984425ed5af89d93ed2f11b6b86020e3457bae21",
			},
		},
		common.HexToAddress(usdcAddr): {
			Token:  usdcAddr,
			Wallet: "0x47F3C2557364EFC28f1269e3169773fa5236384D",
			Holders: []string{
				"0x0000000000000000000000000000000000000000",
				"0xe86a549cebab14ddff9741fd46e62a60ebff5b23",
			},
		},
	}
)

func TestCalcAmountOutAEVMWithUSDCE_USDCPoolWithGRPCClient(t *testing.T) {
	t.Skip()

	client, err := aevmclient.NewGRPCClient(aevmServerURL)
	require.NoError(t, err)

	stateRoot, err := client.LatestStateRoot(context.Background())
	require.NoError(t, err)

	names := []string{"without holders lists", "with holders lists"}
	for i, balanceSlots := range []map[common.Address]*routerentity.ERC20BalanceSlot{balanceSlots, balanceSlotsWithHoldersList} {
		balanceSlots := balanceSlots
		t.Run(names[i], func(t *testing.T) {
			p, err := NewPoolAEVM(
				valueobject.ChainIDAvalancheCChain,
				entity.Pool{
					Address: btcbUSDCPool,
					Tokens: []*entity.PoolToken{
						{Address: btcbAddr},
						{Address: usdcAddr},
					},
					Reserves: entity.PoolReserves{"0", "0"},
				},
				client,
				common.Hash(stateRoot),
				balanceSlots,
			)
			require.NoError(t, err)
			result, _, err := p.CalcAmountOutAEVM(
				pool.TokenAmount{
					Token:  btcbAddr,
					Amount: big.NewInt(1_000_000_00), // 1 BTC
				},
				usdcAddr,
				false,
			)
			require.NoError(t, err)
			fmt.Printf("swapping 1 BTC.b for USDC amountOut = %s, gas used = %d\n", result.TokenAmountOut.Amount, result.Gas)
			usdcOut := new(big.Int).Set(result.TokenAmountOut.Amount)

			p.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: result.SwapInfo})

			result, _, err = p.CalcAmountOutAEVM(
				pool.TokenAmount{
					Token:  usdcAddr,
					Amount: usdcOut,
				},
				btcbAddr,
				false,
			)
			require.NoError(t, err)
			fmt.Printf("swapping %s USDC for BTC.b amountOut = %s, gas used = %v\n", usdcOut, result.TokenAmountOut.Amount, result.Gas)
		})
	}
}
