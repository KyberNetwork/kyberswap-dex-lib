package arcade

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// maxDriftUnits bounds the drift of the shared v3-derived swap math against the deployed
// contracts, the same inherited effect documented in hooks/b20: measured at 0 to 23 base
// units across these fixtures, never proportional to size, and always on the low side
// (the simulator never quotes more than the pool pays). It is not this hook's fee math:
// isolating the raw v3 output and applying the hook fee by hand reproduces the same
// offset, and the fee itself is an exact floor mulDiv like the contract's.
const maxDriftUnits = 25

func assertWithinDrift(t *testing.T, want string, got *big.Int) {
	t.Helper()
	wantBI, ok := new(big.Int).SetString(want, 10)
	require.True(t, ok)
	require.LessOrEqualf(t, got.Cmp(wantBI), 0, "simulator quoted more than the contract pays: want=%s got=%s", wantBI, got)
	diff := new(big.Int).Sub(wantBI, got)
	require.LessOrEqualf(t, diff.Int64(), int64(maxDriftUnits), "want=%s got=%s diff=%s", wantBI, got, diff)
}

// TestDirectProtocolMatch_ClankerLaunch: the local simulator's CalcAmountOut must agree
// with Uniswap's canonical V4Quoter on Arc (0x8dc178eF...), which executes the real
// ArcadeHook through PoolManager.unlock, for a live CLANKER launch. Pool state (slot0,
// liquidity and the single locked tick band) and the expected outputs were read at Arc
// block 21002800 on 2026-09-15 via StateView (0xf3334192...) and quoteExactInputSingle.
func TestDirectProtocolMatch_ClankerLaunch(t *testing.T) {
	t.Parallel()

	poolID := "0xe9ab261f1c77caeefd37aec54859484c93c8856b95739e779e4827616cab54fe"
	usdc := "0x3600000000000000000000000000000000000000"
	token := "0x43caace3d7bc72b25e32d2b81b1ee28f447e4ffd"

	staticExtra := `{"0x0":[false,false],"fee":10000,"tS":200,"hooks":"` + HookAddresses[0].Hex() +
		`","uR":"0x4fca4a51ab4f23a7447b3284fbd7d73289a89fb1","pm2":"0x000000000022D473030F116dDEE9F6B43aC78BA3","mc3":"0xcA11bde05977b3631167028862bE2a173976CA11"}`

	extra := `{"liquidity":5954890256379054518,"sqrtPriceX96":13304717705206089269209816145461188496,"tickSpacing":200,"tick":378799,` +
		`"ticks":[{"index":-887200,"liquidityGross":5954890256379054518,"liquidityNet":5954890256379054518},` +
		`{"index":378800,"liquidityGross":5954890256379054518,"liquidityNet":-5954890256379054518}],` +
		`"hX":{"tr":true,"m":1,"s":2,"u0":true,"q0":true,"la":1789446515,"bc":true}}`

	pool := entity.Pool{
		Address:     poolID,
		Exchange:    string(valueobject.ExchangeUniswapV4Arcade),
		Type:        uniswapv4.DexType,
		SwapFee:     10000,
		Tokens:      []*entity.PoolToken{{Address: usdc, Swappable: true}, {Address: token, Swappable: true}},
		Reserves:    entity.PoolReserves{"1000000000000000", "1000000000000000000000000000"},
		StaticExtra: staticExtra,
		Extra:       extra,
	}

	// The launch (1789446515) is past its five-minute buy-cap window for this fixture.
	orig := NowFn
	NowFn = func() int64 { return time.Now().Unix() }
	defer func() { NowFn = orig }()

	sim, err := uniswapv4.NewPoolSimulator(pool, valueobject.ChainIDArc)
	require.NoError(t, err)

	buyCases := []struct{ name, amountIn, wantOut string }{
		{"buy 1 USDC", "1000000", "27917416838361141584416"},
		{"buy 50 USDC", "50000000", "1393963963449358840418985"},
		{"buy 123.456789 USDC", "123456789", "3434851982457937035593134"},
		{"buy 1000 USDC", "1000000000", "27159939448547919886904842"},
		{"buy 10000 USDC", "10000000000", "218250372944272014495695336"},
	}
	for _, c := range buyCases {
		t.Run(c.name, func(t *testing.T) {
			amt, _ := new(big.Int).SetString(c.amountIn, 10)
			res, err := sim.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{Token: usdc, Amount: amt},
				TokenOut:      token,
			})
			require.NoError(t, err)
			assertWithinDrift(t, c.wantOut, res.TokenAmountOut.Amount)
		})
	}

	sellCases := []struct{ name, amountIn, wantOut string }{
		{"sell 1 token", "1000000000000000000", "35"},
		{"sell 10 tokens", "10000000000000000000", "351"},
	}
	for _, c := range sellCases {
		t.Run(c.name, func(t *testing.T) {
			amt, _ := new(big.Int).SetString(c.amountIn, 10)
			res, err := sim.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{Token: token, Amount: amt},
				TokenOut:      usdc,
			})
			require.NoError(t, err)
			assertWithinDrift(t, c.wantOut, res.TokenAmountOut.Amount)
		})
	}
}

// TestDirectProtocolMatch_GraduatedPump covers the path where ArcadeHook itself takes the
// fee (a graduated PUMP pool, LP fee 0): on the USDC input of a buy (beforeSwap) and on
// the USDC output of a sell (afterSwap), at a dynamic fee strictly between its bounds.
// Fixtures come from executing the deployed ArcadeHook bytecode against a real v4-core
// PoolManager in Foundry (contracts/v4test, ArcadeHookSwap harness): a PUMP launch
// graduated into its full-range locked position, its fee oracle moved by swaps across
// time, then each exact-in swap executed from a state snapshot. No PUMP launch has
// graduated on Arc mainnet yet, so this is the only source of real hook-fee outputs.
// Both currency orderings, since the fee side depends on which currency USDC is.
func TestDirectProtocolMatch_GraduatedPump(t *testing.T) {
	t.Parallel()

	type swapCase struct{ name, amountIn, wantOut string }
	fixtures := []struct {
		name, poolID, usdc, token, sqrtPriceX96, extraHook string
		tick                                               int
		usdcIsCurrency0                                    bool
		buys, sells                                        []swapCase
	}{
		{
			name: "usdc-currency1", poolID: "0xc2faa2d5a317323e165259d900878159d6ef1784dfac15c06e3e33ebcba70a7f",
			usdc: "0x2e234dae75c793f67a35089c9d99245e1c58470b", token: "0x0fe35c3d33fddc8c8dcaf271efcea03251d81fc4",
			sqrtPriceX96: "4713501892870190559539", tick: -332765, usdcIsCurrency0: false,
			// emaTickE3 -366089500, gradMcapTick -373573: growth 7484 ticks, fee 78 bps.
			extraHook: `{"tr":true,"m":0,"s":2,"oi":true,"ema":-366089500,"gt":-373573}`,
			buys: []swapCase{
				{"buy 1 USDC", "1000000", "280328219618577721584"},
				{"buy 250 USDC", "250000000", "69913711025872150725865"},
				{"buy 5000 USDC", "5000000000", "1337008312859997519606807"},
				{"buy 40000 USDC", "40000000000", "8085622338153118305351229"},
			},
			sells: []swapCase{
				{"sell 1 token", "1000000000000000000", "3512"},
				{"sell 250k tokens", "250000000000000000000000", "870438591"},
				{"sell 20M tokens", "20000000000000000000000000", "41561503361"},
			},
		},
		{
			name: "usdc-currency0", poolID: "0x757f660582e56bd5628ac095a88a3a598b838511a148988963947b304ada69e4",
			usdc: "0x0000000000000000000000000000000000007770", token: "0x0fe35c3d33fddc8c8dcaf271efcea03251d81fc4",
			sqrtPriceX96: "1331727848649649629133391735388252857", tick: 332764, usdcIsCurrency0: true,
			extraHook: `{"tr":true,"m":0,"s":2,"u0":true,"oi":true,"ema":-366088500,"gt":-373572}`,
			buys: []swapCase{
				{"buy 1 USDC", "1000000", "280328219619594993598"},
				{"buy 250 USDC", "250000000", "69913711026125552935909"},
				{"buy 5000 USDC", "5000000000", "1337008312864737469308575"},
				{"buy 40000 USDC", "40000000000", "8085622338178368001001782"},
			},
			sells: []swapCase{
				{"sell 1 token", "1000000000000000000", "3512"},
				{"sell 250k tokens", "250000000000000000000000", "870438591"},
				{"sell 20M tokens", "20000000000000000000000000", "41561503361"},
			},
		},
	}

	for _, f := range fixtures {
		t.Run(f.name, func(t *testing.T) {
			tokens := []*entity.PoolToken{{Address: f.token, Swappable: true}, {Address: f.usdc, Swappable: true}}
			if f.usdcIsCurrency0 {
				tokens[0], tokens[1] = tokens[1], tokens[0]
			}
			staticExtra := `{"0x0":[false,false],"fee":0,"tS":200,"hooks":"` + HookAddresses[0].Hex() +
				`","uR":"0x4fca4a51ab4f23a7447b3284fbd7d73289a89fb1","pm2":"0x000000000022D473030F116dDEE9F6B43aC78BA3","mc3":"0xcA11bde05977b3631167028862bE2a173976CA11"}`
			extra := `{"liquidity":1724627447676164059,"sqrtPriceX96":` + f.sqrtPriceX96 + `,"tickSpacing":200,"tick":` +
				big.NewInt(int64(f.tick)).String() + `,` +
				`"ticks":[{"index":-887200,"liquidityGross":1724627447676164059,"liquidityNet":1724627447676164059},` +
				`{"index":887200,"liquidityGross":1724627447676164059,"liquidityNet":-1724627447676164059}],` +
				`"hX":` + f.extraHook + `}`

			sim, err := uniswapv4.NewPoolSimulator(entity.Pool{
				Address:     f.poolID,
				Exchange:    string(valueobject.ExchangeUniswapV4Arcade),
				Type:        uniswapv4.DexType,
				SwapFee:     0,
				Tokens:      tokens,
				Reserves:    entity.PoolReserves{"1000000000000000000000000000", "1000000000000000000000000000"},
				StaticExtra: staticExtra,
				Extra:       extra,
			}, valueobject.ChainIDArc)
			require.NoError(t, err)

			for _, c := range f.buys {
				amt, _ := new(big.Int).SetString(c.amountIn, 10)
				res, err := sim.CalcAmountOut(poolpkg.CalcAmountOutParams{
					TokenAmountIn: poolpkg.TokenAmount{Token: f.usdc, Amount: amt},
					TokenOut:      f.token,
				})
				require.NoError(t, err, c.name)
				assertWithinDrift(t, c.wantOut, res.TokenAmountOut.Amount)
			}
			for _, c := range f.sells {
				amt, _ := new(big.Int).SetString(c.amountIn, 10)
				res, err := sim.CalcAmountOut(poolpkg.CalcAmountOutParams{
					TokenAmountIn: poolpkg.TokenAmount{Token: f.token, Amount: amt},
					TokenOut:      f.usdc,
				})
				require.NoError(t, err, c.name)
				assertWithinDrift(t, c.wantOut, res.TokenAmountOut.Amount)
			}
		})
	}
}
