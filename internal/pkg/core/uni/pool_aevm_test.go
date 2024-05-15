package uni

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	aevmclient "github.com/KyberNetwork/aevm/client"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	routerentity "github.com/KyberNetwork/router-service/internal/pkg/entity"
)

const (
	aevmServerURL = "localhost:8246" // CHANGE THIS

	usdtAddr = "0xdac17f958d2ee523a2206206994597c13d831ec7"
	wethAddr = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
	wbtcAddr = "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599"
	daiAddr  = "0x6b175474e89094c44da98b954eedeac495271d0f"
	gst2Addr = "0x0000000000b3f879cb30fe243b4dfee438691c04"
	chiAddr  = "0x0000000000004946c0e9f43f4dee607b0ef1fa1c"
	usdcAddr = "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
	ddaiAddr = "0x00000000001876eb1444c986fd502e618c587430"

	usdtWETHPoolAddr = "0x0d4a11d5EEaaC28EC3F61d100daF4d40471f1852"
	wbtcDAIPoolAddr  = "0x231b7589426ffe1b75405526fc32ac09d44364c4"
	gst2WETHPoolAddr = "0x27c64bdca05d79f6ee32c3e981dc5153d9d794cd"
	gst2DAIPoolAddr  = "0x2bb8c3d1cc99f4592211424ee3dd1463d2be0f7e"
	chiWETHPoolAddr  = "0xa6f3ef841d371a82ca757fad08efc0dee2f1f5e2"
	daiUSDCPoolAddr  = "0xae461ca67b15dc8dc81ce7615e0320da1a9ab8d5"
	usdcWETHPoolAddr = "0xb4e16d0168e52d35cacd2c6185b44281ec28c9dc"
	wbtcWETHPoolAddr = "0xbb2b8038a1640196fbe3e38816f3e67cba72d940"
	ddaiDAIPoolAddr  = "0xce26a65e7ad8c24589410e3348f4392635ce4172"
)

var (
	tokenNames = map[string]string{
		usdtAddr: "USDT",
		wethAddr: "WETH",
		wbtcAddr: "WBTC",
		daiAddr:  "DAI",
		gst2Addr: "GST2",
		chiAddr:  "CHI",
		usdcAddr: "USDC",
		ddaiAddr: "dDAI",
	}
	tokenDecimals = map[string]int{
		wethAddr: 18,
		wbtcAddr: 8,
		usdcAddr: 6,
		usdtAddr: 6,
		daiAddr:  18,
		chiAddr:  0,
		gst2Addr: 2,
	}

	balanceSlots = map[common.Address]*routerentity.ERC20BalanceSlot{
		common.HexToAddress(usdtAddr): {
			Token:       usdtAddr,
			Wallet:      "0x19767032471665DF0FD7f6160381a103eCe6261A",
			BalanceSlot: "0x2ac6b3a830729524228bab8903e26d3fbed6c148c8fe764a4c64da421d3fa3a5",
		},
		common.HexToAddress(wethAddr): {
			Token:       wethAddr,
			Wallet:      "0x19767032471665DF0FD7f6160381a103eCe6261A",
			BalanceSlot: "0xd82bd39a7a5d839840e3e8e4207676dd90b9b46061977888df793fe145ec3c9e",
		},
		common.HexToAddress(wbtcAddr): {
			Token:       wbtcAddr,
			Wallet:      "0x19767032471665df0fd7f6160381a103ece6261a",
			BalanceSlot: "0x4481c9258c8aa2f465a384b26482a9e93c21827d99cd79f2a29bba446feb0f92",
		},
		common.HexToAddress(daiAddr): {
			Token:       daiAddr,
			Wallet:      "0x19767032471665DF0FD7f6160381a103eCe6261A",
			BalanceSlot: "0x2ac6b3a830729524228bab8903e26d3fbed6c148c8fe764a4c64da421d3fa3a5",
		},
		common.HexToAddress(gst2Addr): {
			Token:       gst2Addr,
			Wallet:      "0x19767032471665DF0FD7f6160381a103eCe6261A",
			BalanceSlot: "0x4481c9258c8aa2f465a384b26482a9e93c21827d99cd79f2a29bba446feb0f92",
		},
		common.HexToAddress(chiAddr): {
			Token:       chiAddr,
			Wallet:      "0x19767032471665DF0FD7f6160381a103eCe6261A",
			BalanceSlot: "0x4481c9258c8aa2f465a384b26482a9e93c21827d99cd79f2a29bba446feb0f92",
		},
		common.HexToAddress(usdcAddr): {
			Token:       usdcAddr,
			Wallet:      "0x19767032471665DF0FD7f6160381a103eCe6261A",
			BalanceSlot: "0x43089540b2bb76ddad9b8269cb1c4d44387eaf1402cb2d4c55384a2542f56c6b",
		},
	}
)

var (
	routerAddress = common.HexToAddress("0x7a250d5630b4cf539739df2c5dacb4c659f2488d")
)

type testcase struct {
	testName string
	poolAddr string
	tokenIn  string
	tokenOut string
	amountIn *big.Int
}

func amountUI(rawAmount *big.Int, decimal int) *big.Float {
	return new(big.Float).Quo(
		new(big.Float).SetInt(rawAmount),
		new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimal)), nil)),
	)
}

func doTestCalcAmountOut(t *testing.T, client aevmclient.Client, c testcase) {
	stateRoot, err := client.LatestStateRoot(context.Background())
	require.NoError(t, err)

	p, err := NewPoolAEVM(
		entity.Pool{
			Address:  c.poolAddr,
			Tokens:   []*entity.PoolToken{{}, {}},
			Reserves: []string{"", ""},
		},
		routerAddress,
		client,
		common.Hash(stateRoot),
		balanceSlots,
	)
	require.NoError(t, err)
	result, err := p.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  c.tokenIn,
			Amount: new(big.Int).Set(c.amountIn),
		},
		TokenOut: c.tokenOut,
	})
	require.NoError(t, err)
	fmt.Printf(
		"swapping %s %s for %s amountOut = %s\n",
		amountUI(c.amountIn, tokenDecimals[c.tokenIn]).String(),
		tokenNames[c.tokenIn], tokenNames[c.tokenOut],
		amountUI(result.TokenAmountOut.Amount, tokenDecimals[c.tokenOut]).String(),
	)

	p.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: result.SwapInfo})

	result, err = p.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  c.tokenIn,
			Amount: new(big.Int).Set(c.amountIn),
		},
		TokenOut: c.tokenOut,
	})
	require.NoError(t, err)
	fmt.Printf(
		"swapping %s %s (again) for %s amountOut = %s\n",
		amountUI(c.amountIn, tokenDecimals[c.tokenIn]).String(),
		tokenNames[c.tokenIn], tokenNames[c.tokenOut],
		amountUI(result.TokenAmountOut.Amount, tokenDecimals[c.tokenOut]).String(),
	)
	amountOut := new(big.Int).Set(result.TokenAmountOut.Amount)

	p.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: result.SwapInfo})

	result, err = p.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  c.tokenOut,
			Amount: amountOut,
		},
		TokenOut: c.tokenIn,
	})
	require.NoError(t, err)
	fmt.Printf(
		"swapping %s %s for %s amountOut = %s gas = %v\n",
		amountUI(amountOut, tokenDecimals[c.tokenOut]).String(),
		tokenNames[c.tokenIn], tokenNames[c.tokenOut],
		amountUI(result.TokenAmountOut.Amount, tokenDecimals[c.tokenIn]).String(),
		result.Gas,
	)
}

func TestCalcAmountOutWithGRPCClient(t *testing.T) {
	t.Skip()

	testcases := []testcase{
		{
			testName: "USDT-WETH",
			poolAddr: usdtWETHPoolAddr,
			tokenIn:  usdtAddr,
			tokenOut: wethAddr,
			amountIn: big.NewInt(500_000 * 1_000_000), // 500K USDT
		},
		{
			testName: "WBTC-DAI",
			poolAddr: wbtcDAIPoolAddr,
			tokenIn:  daiAddr,
			tokenOut: wbtcAddr,
			amountIn: new(big.Int).Mul(big.NewInt(500), big.NewInt(1_000_000_000_000_000_000)), // 500 DAI
		},
		{
			testName: "GST2-WETH",
			poolAddr: gst2WETHPoolAddr,
			tokenIn:  gst2Addr,
			tokenOut: wethAddr,
			amountIn: new(big.Int).Mul(big.NewInt(10), big.NewInt(100)), // 10 GST2,
		},
		{
			testName: "GST2-DAI",
			poolAddr: gst2DAIPoolAddr,
			tokenIn:  gst2Addr,
			tokenOut: daiAddr,
			amountIn: new(big.Int).Mul(big.NewInt(10), big.NewInt(100)), // 10 GST2
		},
		{
			testName: "CHI-WETH",
			poolAddr: chiWETHPoolAddr,
			tokenIn:  chiAddr,
			tokenOut: wethAddr,
			amountIn: big.NewInt(5), // 5 CHI
		},
		{
			testName: "DAI-USDC",
			poolAddr: daiUSDCPoolAddr,
			tokenIn:  usdcAddr,
			tokenOut: daiAddr,
			amountIn: new(big.Int).Mul(big.NewInt(500), big.NewInt(1_000_000)), // 500 USDC
		},
		{
			testName: "USDC-WETH",
			poolAddr: usdcWETHPoolAddr,
			tokenIn:  usdcAddr,
			tokenOut: wethAddr,
			amountIn: new(big.Int).Mul(big.NewInt(500), big.NewInt(1_000_000)),
		},
		{
			testName: "WBTC-WETH",
			poolAddr: wbtcWETHPoolAddr,
			tokenIn:  wethAddr,
			tokenOut: wbtcAddr,
			amountIn: new(big.Int).Mul(big.NewInt(1), big.NewInt(1_000_000_000_000_000_000)), // 1 ETH
		},
	}
	for _, c := range testcases {
		t.Run(c.testName, func(t *testing.T) {
			client, err := aevmclient.NewGRPCClient(aevmServerURL)
			require.NoError(t, err)
			doTestCalcAmountOut(t, client, c)
		})
	}
}

/*
0x026babd2ae9379525030fc2574e39bc156c10583
    WTBC -> USDC: OK
    USDC -> WBTC: eth_call RPC method also returns "execution reverted"

0x0d4a11d5eeaac28ec3f61d100daf4d40471f1852
    USDT/WETH pool, successfully tested in core/uni/pool_aevm_test.go

0x11886971af09bc9a3d20e8d0ec3e9af7ee7a214d
    UOU/WETH pool with only 2.6e-17 WETH, drained pool

0x1e8f1568b598908785064809ebf5745004ce3962
    TUSDT/USDT pool, drained

0x231b7589426ffe1b75405526fc32ac09d44364c4
    WBTC/DAI pool, successfully tested in core/uni/pool_aevm_test.go

0x27c64bdca05d79f6ee32c3e981dc5153d9d794cd
    GST2/WETH pool, successfully tested in core/uni/pool_aevm_test.go

0x2906fdf9c18bf42e950f78b9ac210934c1e93de8
    TUSD/USDT pool, drained

0x2bb8c3d1cc99f4592211424ee3dd1463d2be0f7e
    GST2/DAI pool, successfully tested in core/uni/pool_aevm_test.go

0x2fc246a27a65c0ad8caa5ef41aa2ee0a8449e9b1
    STRIP/WETH pool, drained

0x5a59e4e647a3acc42b01715f3a1d271c1f7e7aeb
    WBTC/USDT pool, drained

0xa6f3ef841d371a82ca757fad08efc0dee2f1f5e2
    CHI/WETH pool, successfully tested in core/uni/pool_aevm_test.go

0xa93eb5b410b651514a18724872306f5ce9928dde
    WBTC/DAI pool, drained

0xadddab5f35baee3c7923485e390e92e0df82f1c1
    CHI/WETH pool, drained

0xae461ca67b15dc8dc81ce7615e0320da1a9ab8d5
    DAI/USDC pool, successfully tested in core/uni/pool_aevm_test.go

0xb4e16d0168e52d35cacd2c6185b44281ec28c9dc
    USDC/WETH pool, successfully tested in core/uni/pool_aevm_test.go

0xbb2b8038a1640196fbe3e38816f3e67cba72d940
    WBTC/WETH pool, successfully tested in core/uni/pool_aevm_test.go

0xc33dcd77650ae2382665be0dda21eb6d4da37cda
    BID/WETH pool, drained

0xce26a65e7ad8c24589410e3348f4392635ce4172
    dDAI/DAI pool, successfully tested in core/uni/pool_aevm_test.go

0xd13f52c62f2fd4b4b042b4bef84fd08f8d2054cf
    CHI/DAI pool, drained

0xd4bda8d3f3fa647caefad00032d0908547e00e9c
    TUSD/USDT pool, drained

0xe55e68925809784c8234dfcf6f8fa42c3a48b2c3
    TUSD/USDC pool, drained

0xf284aae87a0f10c6aa5eaf51eadcb2d736a448d9
    LON/WETH pool, drained

0xf5148fbdae394c553d019b4caeffc5f845dcd12c
    TUSD/USDC pool, drained
*/
