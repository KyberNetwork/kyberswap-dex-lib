package brownfiv3

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// On-chain data sourced from Berachain block 21048469.
//
// Pair 0 — WETH/USDC (0x3E6200Dc34C3b5967E7bBdCf5FA74153348E9694)
//
//	token0 = WETH (18 dec), token1 = USDC (6 dec), quoteTokenIndex = 1 (token0 = base)
//	reserves: r0=2968503755735635, r1=5797793
//	factory.getSwapPrices returns:
//	  pythPrice0 = 39195766625498220740512  (WETH Q64 dollar price)
//	  pythPrice1 = 18443500582699071265     (USDC Q64 dollar price)
//	  ammPrice   = 39214718855475040805758  (on-chain AMM relative price Q64)
//	  adjPrice   = 39208689242307926688245  (weighted 50/50 mean Q64)
//	  sPrice0_sell = 39205980540022840550524  (WETH price for SELL direction)
//	  sPrice0_buy  = 39198140127956042662204  (WETH price for BUY direction)
var (
	poolWETHUSDC entity.Pool
	_            = json.Unmarshal([]byte(`{
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
				"100000000000":    "211",
				"468661346370449": "991872",
				// 0.01 WETH exceeds reserves → cutoff
				"10000000000000000": "CUTOFF_INPUT_LIMIT_REACHED",
			},
		},
		// token1 (USDC) → token0 (WETH): SELL direction, isSell=true
		1: {
			0: {
				// 1 USDC → ~0.000469 WETH (fits within ~5.8 USDC reserves)
				"1000000": "468661335135063",
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

func TestCloneState(t *testing.T) {
	t.Parallel()
	testutil.TestCloneState(t, simWETHUSDC, pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0x2f6f07cdcf3588944bf4c42ac74ff24bf56e7590", Amount: big.NewInt(468661346370449)},
		TokenOut:      "0x549943e04f40284185054145c6e4e9568c1d3241",
	}, nil)
}

// New factory (0x6Ccf36d3...) WETH/USDC.e pool — block 21995373, Berachain.
var (
	poolWETHUSDCNew entity.Pool
	_               = json.Unmarshal([]byte(`{
		"address":     "0xc123bc9259d1a99add5a2c512498ac146dd2bade",
		"exchange":    "brownfi-v3",
		"type":        "brownfi-v3",
		"timestamp":   1999999999,
		"blockNumber": 21995373,
		"reserves":    ["296881910284765994","490783665"],
		"tokens": [
			{"address":"0x2f6f07cdcf3588944bf4c42ac74ff24bf56e7590","decimals":18,"swappable":true},
			{"address":"0x549943e04f40284185054145c6e4e9568c1d3241","decimals":6, "swappable":true}
		],
		"extra": "{\"kB\":\"184467440737095516\",\"kQ\":\"184467440737095516\",\"f\":300000,\"g\":80000000,\"l\":36893488147419103,\"fs\":10000,\"pw\":50000000,\"dt\":300000,\"p0\":\"30821262219198118827300\",\"p1\":\"18442135339170176021\",\"c0\":\"11345248618900416195\",\"c1\":\"9312469810730792\",\"am\":\"30820011453736467775190\"}",
		"staticExtra": "{\"pf\":[\"0xff61491a931112ddf1bd8147cd1b641375f79f5825126d665480874634fd0ace\",\"0xeaa020c61cc479712813461ce153894a96a6c00b21ed0cfc2798d1f9a9e9c94a\"],\"po\":\"0x315062ea5686289bcbe138424fd10591beb37a75\",\"pc\":\"0x4955e0d8a7f25ba83216946c17fe791d8c49c43a\",\"qi\":1,\"lu\":1781005072}"
	}`), &poolWETHUSDCNew)

	simWETHUSDCNew = lo.Must(NewPoolSimulator(pool.FactoryParams{
		EntityPool: poolWETHUSDCNew,
		ChainID:    valueobject.ChainIDBerachain,
	}))
)

// verifyOutputAchievable asserts that CalcAmountOut returns a positive amount and that
// the binary-search result is both achievable (CalcAmountIn(out) <= amountIn) and maximal
// (CalcAmountIn(out+1) > amountIn or returns an error).
func verifyOutputAchievable(t *testing.T, sim *PoolSimulator, tokenIn, tokenOut string, amountIn *big.Int) {
	t.Helper()
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amountIn},
		TokenOut:      tokenOut,
	})
	require.NoError(t, err)
	out := res.TokenAmountOut.Amount
	require.True(t, out.Sign() > 0, "output must be positive")

	// Achievability: CalcAmountIn(out) must not exceed amountIn.
	inRes, err := sim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: tokenOut, Amount: out},
		TokenIn:        tokenIn,
	})
	require.NoError(t, err, "CalcAmountIn(out) must not error")
	assert.True(t, inRes.TokenAmountIn.Amount.Cmp(amountIn) <= 0,
		"CalcAmountIn(%s) = %s > amountIn %s: output is not achievable",
		out, inRes.TokenAmountIn.Amount, amountIn)

	// Maximality: CalcAmountIn(out+1) must exceed amountIn.
	outPlus1 := new(big.Int).Add(out, big.NewInt(1))
	inResPlus1, errPlus1 := sim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: tokenOut, Amount: outPlus1},
		TokenIn:        tokenIn,
	})
	if errPlus1 == nil {
		assert.True(t, inResPlus1.TokenAmountIn.Amount.Cmp(amountIn) > 0,
			"CalcAmountIn(%s) = %s <= amountIn %s: output is not maximal",
			outPlus1, inResPlus1.TokenAmountIn.Amount, amountIn)
	}
}

// TestVerify_SearchInvariant_WETHUSDCNew checks the binary-search achievability/maximality
// invariant for the new-factory WETH/USDC.e pool in both swap directions.
func TestVerify_SearchInvariant_WETHUSDCNew(t *testing.T) {
	t.Parallel()

	weth := "0x2f6f07cdcf3588944bf4c42ac74ff24bf56e7590"
	usdc := "0x549943e04f40284185054145c6e4e9568c1d3241"

	// BUY: WETH → USDC.e (token0 → token1, isSell=false)
	for _, amtWEI := range []int64{1e9, 1e11, 1e13, 1e14} {
		verifyOutputAchievable(t, simWETHUSDCNew, weth, usdc, big.NewInt(amtWEI))
	}

	// SELL: USDC.e → WETH (token1 → token0, isSell=true)
	for _, amtUSDC := range []int64{1e4, 1e5, 1e6, 1e7} {
		verifyOutputAchievable(t, simWETHUSDCNew, usdc, weth, big.NewInt(amtUSDC))
	}
}
