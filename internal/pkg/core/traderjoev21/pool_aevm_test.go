package traderjoev21

import (
	"fmt"
	"math/big"
	"runtime"
	"testing"

	aevmclient "github.com/KyberNetwork/aevm/client"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"

	routerentity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
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
)

func TestCalcAmountOutAEVMWithUSDCE_USDCPoolWithTCPClient(t *testing.T) {
	client, err := aevmclient.NewTCPClient(aevmServerURL, runtime.NumCPU())
	if err != nil {
		t.Skip("could not connect to AEVM server")
		return
	}

	stateRoot, err := client.LatestStateRoot()
	require.NoError(t, err)

	p, err := NewPoolAEVM(
		entity.Pool{
			Address: btcbUSDCPool,
			Tokens: []*entity.PoolToken{
				{Address: btcbAddr},
				{Address: usdcAddr},
			},
		},
		client,
		common.Hash(stateRoot),
		balanceSlots,
	)
	require.NoError(t, err)
	result, err := p.CalcAmountOutAEVM(
		pool.TokenAmount{
			Token:  btcbAddr,
			Amount: big.NewInt(1_000_000_00), // 1 BTC
		},
		usdcAddr,
	)
	require.NoError(t, err)
	fmt.Printf("swapping 1 BTC.b for USDC amountOut = %s, gas used = %d\n", result.TokenAmountOut.Amount, result.Gas)
	usdcOut := new(big.Int).Set(result.TokenAmountOut.Amount)

	p.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: result.SwapInfo})

	result, err = p.CalcAmountOutAEVM(
		pool.TokenAmount{
			Token:  usdcAddr,
			Amount: usdcOut,
		},
		btcbAddr,
	)
	require.NoError(t, err)
	fmt.Printf("swapping %s USDC for BTC.b amountOut = %s, gas used = %v\n", usdcOut, result.TokenAmountOut.Amount, result.Gas)
}
