package brownfiv3

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// On-chain data sourced from Berachain block 21048469.
//
// Pair 0 — WETH/USDC (0x3E6200Dc34C3b5967E7bBdCf5FA74153348E9694)
//   token0 = WETH (18 dec), token1 = USDC (6 dec), quoteTokenIndex = 1 (token0 = base)
//   reserves: r0=2968503755735635, r1=5797793
//   factory.getSwapPrices returns:
//     pythPrice0 = 39195766625498220740512  (WETH Q64 dollar price)
//     pythPrice1 = 18443500582699071265     (USDC Q64 dollar price)
//     ammPrice   = 39214718855475040805758  (on-chain AMM relative price Q64)
//     adjPrice   = 39208689242307926688245  (weighted 50/50 mean Q64)
//     sPrice0_sell = 39205980540022840550524  (WETH price for SELL direction)
//     sPrice0_buy  = 39198140127956042662204  (WETH price for BUY direction)
var (
	poolWETHUSDC entity.Pool
	_ = json.Unmarshal([]byte(`{
		"address":     "0x3e6200dc34c3b5967e7bbdcf5fa74153348e9694",
		"exchange":    "brownfi-v3",
		"type":        "brownfi-v3",
		"timestamp":   1999999999,
		"blockNumber": 21048469,
		"reserves":    ["2968503755735635","5797793"],
		"tokens": [
			{"address":"0x2f6f07cdcf3588944bf4c42ac74ff24bf56e7590","decimals":18,"swappable":true},
			{"address":"0x549943e04f40284185054145c6e4e9568c1d3241","decimals":6, "swappable":true}
		],
		"extra": "{\"kB\":\"184467440737095516\",\"kQ\":\"184467440737095516\",\"f\":300000,\"g\":80000000,\"l\":36893488147419103,\"ss\":0,\"sb\":0,\"fs\":10000,\"cp\":0,\"sbd\":0,\"pw\":50000000,\"dt\":300000,\"p0\":\"39195766625498220740512\",\"p1\":\"18443500582699071265\",\"c0\":\"0\",\"c1\":\"0\",\"am\":\"39214718855475040805758\"}",
		"staticExtra": "{\"f\":[\"0xff61491a931112ddf1bd8147cd1b641375f79f5825126d665480874634fd0ace\",\"0xeaa020c61cc479712813461ce153894a96a6c00b21ed0cfc2798d1f9a9e9c94a\"],\"o\":\"0x538e83408504faa2c97fb12b7ce1f8b6989d8be4\",\"pc\":\"0xc04cd132781b628ce3583d1f949d03db52ba753c\",\"qi\":1,\"lu\":1746000000}"
	}`), &poolWETHUSDC)

	simWETHUSDC = lo.Must(NewPoolSimulator(pool.FactoryParams{
		EntityPool: poolWETHUSDC,
		ChainID:    valueobject.ChainIDBerachain,
	}))
)

// TestCalcAmountOut_WETH_USDC tests swap simulation for both directions.
//
// Pool reserves at block 21048469: r0≈0.003 WETH, r1≈5.8 USDC — very thin.
// Expected values produced by probe_test.go against the same on-chain state.
//
// Effective prices (derived from on-chain getSwapPrices):
//
//	BUY  (WETH→USDC, isSell=false): ~2112.5 USDC/WETH (adjPrice spread-adjusted)
//	SELL (USDC→WETH, isSell=true):  ~2136.0 WETH/USDC (inverse)
func TestCalcAmountOut_WETH_USDC(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountOut(t, simWETHUSDC, map[int]map[int]map[string]string{
		// token0 (WETH) → token1 (USDC): BUY direction, isSell=false
		0: {
			1: {
				// 0.001 WETH → ~2.112 USDC (fits within ~0.003 WETH reserves)
				"1000000000000000": "2112524",
				// 0.01 WETH exceeds reserves → cutoff
				"10000000000000000": "CUTOFF_INPUT_LIMIT_REACHED",
				// 1 WETH exceeds reserves → cutoff
				"1000000000000000000": "CUTOFF_INPUT_LIMIT_REACHED",
			},
		},
		// token1 (USDC) → token0 (WETH): SELL direction, isSell=true
		1: {
			0: {
				// 1 USDC → ~0.000469 WETH (fits within ~5.8 USDC reserves)
				"1000000": "468661346370449",
				// 10 USDC exceeds reserves → cutoff
				"10000000": "CUTOFF_INPUT_LIMIT_REACHED",
				// 100 USDC exceeds reserves → cutoff
				"100000000": "CUTOFF_INPUT_LIMIT_REACHED",
			},
		},
	})
}

func TestCalcAmountIn_WETH_USDC(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountIn(t, simWETHUSDC)
}
