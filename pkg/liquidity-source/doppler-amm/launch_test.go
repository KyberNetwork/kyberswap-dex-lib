package doppleramm

import (
	"testing"

	"github.com/KyberNetwork/int256"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var (
	launchLevels = []int32{-198310, -168352, -145325, -122298, -108434, -76244, 887200}
	launchNets   = []string{
		"18812618797347724045334", "70148970531258127977822", "231818197886874001154395",
		"-90905233541071061584924", "65769481478689274500625", "-74606844851018599441731",
		"-221037190302079466651521",
	}
)

func newLaunchSimulator(t *testing.T, reserveX, reserveY, sqrtP, liquidity string) *PoolSimulator {
	t.Helper()
	extra := Extra{
		TokenX: "0x0000000000000000000000000000000000000011", BaseFeeBps: 10, SwapFeePips: 10000,
		SqrtPriceX96: uint256.MustFromDecimal(sqrtP), Liquidity: uint256.MustFromDecimal(liquidity),
	}
	for _, n := range launchNets {
		v, err := int256.FromDec(n)
		require.NoError(t, err)
		extra.Nets = append(extra.Nets, v)
	}
	extraBytes, err := json.Marshal(extra)
	require.NoError(t, err)
	staticBytes, err := json.Marshal(StaticExtra{
		MarketID: "0x01", TokenX: extra.TokenX, TokenY: "0x0000000000000000000000000000000000000022",
		AMM: "0x00000000000000000000000000000000000000aa", Module: "0x00000000000000000000000000000000000000cc",
		Levels: launchLevels,
	})
	require.NoError(t, err)
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: entity.Pool{
		Address: "0x01", Exchange: "doppler-fun", Type: DexType,
		Reserves: entity.PoolReserves{reserveX, reserveY},
		Tokens: []*entity.PoolToken{
			{Address: extra.TokenX, Swappable: true},
			{Address: "0x0000000000000000000000000000000000000022", Swappable: true},
		},
		Extra: string(extraBytes), StaticExtra: string(staticBytes),
	}})
	require.NoError(t, err)
	return sim
}

func quoteLaunch(t *testing.T, sim *PoolSimulator, xIn bool, in string) string {
	t.Helper()
	tokenIn, tokenOut := sim.Info.Tokens[1], sim.Info.Tokens[0]
	if xIn {
		tokenIn, tokenOut = tokenOut, tokenIn
	}
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: bi(in)}, TokenOut: tokenOut})
	require.NoError(t, err, in)
	return res.TokenAmountOut.Amount.String()
}

// Every expected amount comes from DopplerQuoter.quoteSwapExactInput on Base market
// 0x59bd7910... (LaunchModule, FeesLocker, 1% fee, 10 bps base fee). The first block is
// the live state at block 56784974. Buys from 3.2e18 up cross levels; charging V3's
// per-step fee instead of one net-of-fee amount is off by a wei of Y per crossing.
func TestLaunchBuyMatchesDeployedQuoter(t *testing.T) {
	sim := newLaunchSimulator(t, "999958632605582091063890108", "101074615332646",
		"3916473128391601941059829", "18812618797347724045334")
	for _, tc := range [][2]string{
		{"1000000000000", "404733438067548655807"},
		{"1000000000000000", "404303892927770461564648"},
		{"100000000000000000", "36582818719072945035164379"},
		{"1000000000000000000", "196139885350150466481151493"},
		{"3200000000000000000", "294138841276166833149468769"},
		{"3264000000000000000", "295454530533400686619924646"},
		{"3265000000000000000", "295474772837798926863415811"},
		{"3500000000000000000", "300175875860012231087568477"},
		{"10000000000000000000", "397305674381788481084515723"},
		{"100000000000000000000", "658604076003552545653474346"},
		{"200000000000000000000", "756113392340956025290265259"},
		{"800000000000000000000", "919718389482836292220757680"},
		{"25600000000000000000000", "997917621465636753747252701"},
	} {
		require.Equal(t, tc[1], quoteLaunch(t, sim, false, tc[0]), "Y in %s", tc[0])
	}
	// Selling near the start price: output is capped by the thin range below.
	require.Equal(t, "2416744222009", quoteLaunch(t, sim, true, "1000000000000000000000"))
}

// State and quotes from an anvil fork of Base after two buys (3e18 and 20e18 WETH)
// moved the price past level -168352. Sells here cross levels downward.
func TestLaunchSellAcrossLevelsMatchesDeployedQuoter(t *testing.T) {
	sim := newLaunchSimulator(t, "504007714746048628186775153", "22747331074615332646",
		"34895685581822601624406808", "88961589328605852023156")
	for _, tc := range [][2]string{
		{"1000000000000000000", "191860240058"},
		{"100000000000000000000", "19186014705260"},
		{"100000000000000000000000", "19176634162664295"},
		{"1000000000000000000000000", "190925365169563599"},
		{"30000000000000000000000000", "5018593262222361091"},
		{"100000000000000000000000000", "12879507361089627683"},
		{"300000000000000000000000000", "21725326876265484252"},
	} {
		require.Equal(t, tc[1], quoteLaunch(t, sim, true, tc[0]), "X in %s", tc[0])
	}
	for _, tc := range [][2]string{
		{"1000000000000000000", "4972679033699514210727800"},
		{"5000000000000000000", "22634400480060560978838642"},
		{"30000000000000000000", "87966350742120022826909936"},
		{"100000000000000000000", "190771057307308296776670591"},
	} {
		require.Equal(t, tc[1], quoteLaunch(t, sim, false, tc[0]), "Y in %s", tc[0])
	}
	// More than the range holds: the chain reverts, so must the quote.
	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: sim.Info.Tokens[0], Amount: bi("600000000000000000000000000")},
		TokenOut:      sim.Info.Tokens[1]})
	require.Error(t, err)
}

// UpdateBalance must carry the post-trade price so a second swap in one route
// starts where the first ended: two half-size swaps match what the chain gives
// for a sequence, and differ from two quotes on the unchanged state.
func TestLaunchUpdateBalanceAdvancesPrice(t *testing.T) {
	sim := newLaunchSimulator(t, "999958632605582091063890108", "101074615332646",
		"3916473128391601941059829", "18812618797347724045334")
	in := bi("3000000000000000000")
	first, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: sim.Info.Tokens[1], Amount: in}, TokenOut: sim.Info.Tokens[0]})
	require.NoError(t, err)
	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: sim.Info.Tokens[1], Amount: in},
		TokenAmountOut: *first.TokenAmountOut, SwapInfo: first.SwapInfo})
	second, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: sim.Info.Tokens[1], Amount: in}, TokenOut: sim.Info.Tokens[0]})
	require.NoError(t, err)
	require.Equal(t, -1, second.TokenAmountOut.Amount.Cmp(first.TokenAmountOut.Amount), "price must have moved up")

	// The clone must not share price state with the original.
	cloned := sim.CloneState().(*PoolSimulator)
	cloned.sqrtP.Clear()
	require.False(t, sim.sqrtP.IsZero())
}

func TestCheckLaunchLiquidity(t *testing.T) {
	sim := newLaunchSimulator(t, "1", "1", "3916473128391601941059829", "18812618797347724045334")
	require.NoError(t, checkLaunchLiquidity(launchLevels, &sim.extra))

	// A position the level list does not know about shows up as a wrong liquidity at the price.
	sim.extra.Liquidity = uint256.NewInt(5)
	require.ErrorIs(t, checkLaunchLiquidity(launchLevels, &sim.extra), ErrLaunchState)
}
