package univ3

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	aevmclient "github.com/KyberNetwork/aevm/client"
	jaeger "github.com/KyberNetwork/aevm/types/jaeger"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	aevmcore "github.com/KyberNetwork/router-service/internal/pkg/core/aevm"
	routerentity "github.com/KyberNetwork/router-service/internal/pkg/entity"
)

const (
	aevmServerURL = "localhost:8247" // CHANGE THIS

	usdcAddr = "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
	wethAddr = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"

	lidoAddr = "0x5a98fcbea516cf06857215779fd812ca3bef1b32"
	usdtAddr = "0xdac17f958d2ee523a2206206994597c13d831ec7"

	wbtcAddr = "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599"

	usdc2PoolAddr   = "0x8ad599c3a0ff1de082011efddc58f1908eb6e6d8"
	ldoUSDTPoolAddr = "0x20215Cd3949eDf87771C529eD41F5D8Cce652F65"
)

var (
	routerAddress = common.HexToAddress("0xE592427A0AEce92De3Edee1F18E0157C05861564")

	balanceSlots = map[common.Address]*routerentity.ERC20BalanceSlot{
		common.HexToAddress(usdcAddr): {
			Token:       usdcAddr,
			Wallet:      "0x19767032471665DF0FD7f6160381a103eCe6261A",
			BalanceSlot: "0x43089540b2bb76ddad9b8269cb1c4d44387eaf1402cb2d4c55384a2542f56c6b",
		},
		common.HexToAddress(wethAddr): {
			Token:       wethAddr,
			Wallet:      "0x19767032471665DF0FD7f6160381a103eCe6261A",
			BalanceSlot: "0xd82bd39a7a5d839840e3e8e4207676dd90b9b46061977888df793fe145ec3c9e",
		},
		common.HexToAddress(usdtAddr): {
			Token:       usdtAddr,
			Wallet:      "0x19767032471665DF0FD7f6160381a103eCe6261A",
			BalanceSlot: "0x2ac6b3a830729524228bab8903e26d3fbed6c148c8fe764a4c64da421d3fa3a5",
		},
		common.HexToAddress(wbtcAddr): {
			Token:       wbtcAddr,
			Wallet:      "0x19767032471665df0fd7f6160381a103ece6261a",
			BalanceSlot: "0x4481c9258c8aa2f465a384b26482a9e93c21827d99cd79f2a29bba446feb0f92",
		},
		common.HexToAddress(lidoAddr): {
			Token:       lidoAddr,
			Wallet:      "0x19767032471665DF0FD7f6160381a103eCe6261A",
			BalanceSlot: "0x1d92366e0eb8047c54dfd3b4dbea0f8e23e5b021de7fe643c5e1b27c1a169cf0",
			// the first 128 bits is balance = 0x0000ffffffffffffffffffffffffffff (which is large enough)
			// the last 128 bits is block number at which balance is changed
			PreferredValue: "0x0000ffffffffffffffffffffffffffff000000000000000000000000010c766a",
			ExtraOverrides: map[string]string{
				// balance history length, must be > 0
				"0x5a7df2e997858d39f2aef4861e79c3133f8533d36ea28b6a66fa8b4914908a5a": "0x0000000000000000000000000000000000000000000000000000000000000001",
			},
		},
	}
)

func TestTracingCalcAmountOutAEVMWithUSDC2Pool(t *testing.T) {
	t.Skip()

	client, err := aevmclient.NewGRPCClient(aevmServerURL)
	require.NoError(t, err)
	defer client.Close()

	stateRoot, _ := client.LatestStateRoot(context.Background())

	p, err := NewPoolAEVM(
		entity.Pool{
			Address:  usdc2PoolAddr,
			SwapFee:  3000,
			Tokens:   []*entity.PoolToken{{Address: wethAddr}, {Address: usdcAddr}},
			Reserves: []string{"0", "0"},
		},
		routerAddress,
		1,
		client,
		common.Hash(stateRoot),
		balanceSlots,
	)
	require.NoError(t, err)
	result, traces, err := p.CalcAmountOutAEVM(
		pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  usdcAddr,
				Amount: big.NewInt(500_000_000_000), // 500K USDC
			},
			TokenOut: wethAddr,
		},
		true,
	)
	require.NoError(t, err)
	fmt.Printf("swapping 500K USDC for WETH amountOut = %s\n", result.TokenAmountOut.Amount)

	swapInfo := result.SwapInfo.(*aevmcore.AEVMSwapInfo)
	require.NotNil(t, swapInfo)

	jaegerTraces := jaeger.ToJaegerTrace(traces, "aevm")
	encoded, err := json.MarshalIndent(jaegerTraces, "", "  ")
	require.NoError(t, err)
	fmt.Printf("%s\n", string(encoded))
}

func TestCalcAmountOutAEVMWithUSDC2Pool(t *testing.T) {
	t.Skip()

	client, err := aevmclient.NewGRPCClient(aevmServerURL)
	require.NoError(t, err)
	defer client.Close()

	stateRoot, _ := client.LatestStateRoot(context.Background())

	p, err := NewPoolAEVM(
		entity.Pool{
			Address:  usdc2PoolAddr,
			SwapFee:  3000,
			Tokens:   []*entity.PoolToken{{Address: wethAddr}, {Address: usdcAddr}},
			Reserves: []string{"0", "0"},
		},
		routerAddress,
		1,
		client,
		common.Hash(stateRoot),
		balanceSlots,
	)
	require.NoError(t, err)
	result, err := p.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  usdcAddr,
			Amount: big.NewInt(500_000_000_000), // 500K USDC
		},
		TokenOut: wethAddr,
	})
	require.NoError(t, err)
	fmt.Printf("swapping 500K USDC for WETH amountOut = %s\n", result.TokenAmountOut.Amount)

	p.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: result.SwapInfo})

	result, err = p.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  usdcAddr,
			Amount: big.NewInt(500_000_000_000), // 500K USDC
		},
		TokenOut: wethAddr,
	})
	require.NoError(t, err)
	fmt.Printf("swapping 500K USDC (again) for WETH amountOut = %s\n", result.TokenAmountOut.Amount)
	wethOut := new(big.Int).Set(result.TokenAmountOut.Amount)

	p.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: result.SwapInfo})

	result, err = p.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  wethAddr,
			Amount: wethOut,
		},
		TokenOut: usdcAddr,
	})
	require.NoError(t, err)
	fmt.Printf("swapping %s WETH for USDC amountOut = %s gas = %v\n", wethOut, result.TokenAmountOut.Amount, result.Gas)
}

func TestCalcAmountOutAEVMWithLDOUSDTPool(t *testing.T) {
	t.Skip()

	client, err := aevmclient.NewGRPCClient(aevmServerURL)
	require.NoError(t, err)
	defer client.Close()

	stateRoot, _ := client.LatestStateRoot(context.Background())

	p, err := NewPoolAEVM(
		entity.Pool{
			Address:  ldoUSDTPoolAddr,
			SwapFee:  10000,
			Tokens:   []*entity.PoolToken{{Address: usdtAddr}, {Address: lidoAddr}},
			Reserves: []string{"0", "0"},
		},
		routerAddress,
		1,
		client,
		common.Hash(stateRoot),
		balanceSlots,
	)
	require.NoError(t, err)
	result, err := p.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  lidoAddr,
			Amount: new(big.Int).Mul(big.NewInt(10), big.NewInt(1_000_000_000_000_000_000)),
		},
		TokenOut: usdtAddr,
	})
	require.NoError(t, err)
	fmt.Printf("swapping 10 LIDO for USDT amountOut = %s\n", result.TokenAmountOut.Amount)

	p.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: result.SwapInfo})

	result, err = p.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  lidoAddr,
			Amount: new(big.Int).Mul(big.NewInt(10), big.NewInt(1_000_000_000_000_000_000)),
		},
		TokenOut: usdtAddr,
	})
	require.NoError(t, err)
	fmt.Printf("swapping 10 LIDO (again) for USDT amountOut = %s\n", result.TokenAmountOut.Amount)
	usdtOut := new(big.Int).Set(result.TokenAmountOut.Amount)

	p.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: result.SwapInfo})

	result, err = p.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  usdtAddr,
			Amount: usdtOut,
		},
		TokenOut: lidoAddr,
	})
	require.NoError(t, err)
	fmt.Printf("swapping %s USDT for LIDO amountOut = %s gas = %v\n", usdtOut, result.TokenAmountOut.Amount, result.Gas)
}
