package lotflow

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
)

// F-N2: dex-lib's v3 engine loops forever on a leg pool whose tick disagrees with its sqrtPrice and
// whose tick list repeats an index (FuzzHookExtra/4338fdba81a82b87). validatePool rejects every leg
// state a chain pool cannot be in.
func TestValidatePoolTicks(t *testing.T) {
	tick := func(i int, net int64) uniswapv3.TickU256 {
		g := net
		if g < 0 {
			g = -g
		}
		return uniswapv3.TickU256{Index: i, LiquidityGross: uint256.NewInt(uint64(g)), LiquidityNet: int256.NewInt(net)}
	}
	at := func(i int) *big.Int { s, err := SqrtPriceAtTick(i); require.NoError(t, err); return s }
	good := func() *LegPool {
		return &LegPool{Kind: KindV3, SqrtPriceX96: new(big.Int).Add(at(100), big.NewInt(1)), Tick: 100,
			Liquidity: big.NewInt(1000), TickSpacing: 60,
			Ticks: []uniswapv3.TickU256{tick(-120, 400), tick(60, 600), tick(120, -600), tick(240, -400)}}
	}
	require.NoError(t, validatePool(good()))
	boundary := good()
	boundary.SqrtPriceX96, boundary.Tick = at(120), 119 // a downward stop on tick 120
	require.NoError(t, validatePool(boundary))
	cases := map[string]func(p *LegPool){
		"tick far from sqrtPrice":  func(p *LegPool) { p.Tick = 0 },
		"tick one above":           func(p *LegPool) { p.Tick = 101 },
		"tick one below off-bound": func(p *LegPool) { p.Tick = 99 },
		"duplicate tick":           func(p *LegPool) { p.Ticks[2].Index = 60 },
		"descending ticks":         func(p *LegPool) { p.Ticks[0], p.Ticks[1] = p.Ticks[1], p.Ticks[0] },
		"off spacing":              func(p *LegPool) { p.Ticks[3].Index = 250 },
		"beyond MaxTick":           func(p *LegPool) { p.Ticks[3].Index = (uniswapv3.MaxTick/60 + 1) * 60 },
		"negative liquidity above": func(p *LegPool) { p.Ticks[2] = tick(120, -2000) },
		"negative liquidity below": func(p *LegPool) { p.Ticks[0] = tick(-120, 1100) },
		"liquidity above uint128":  func(p *LegPool) { p.Ticks[3].LiquidityNet = int256.MustFromBig(new(big.Int).Lsh(big.NewInt(1), 128)) },
		"gross nil":                func(p *LegPool) { p.Ticks[1].LiquidityGross = nil },
	}
	for name, mutate := range cases {
		p := good()
		mutate(p)
		require.ErrorIs(t, validatePool(p), errInvalid, name)
	}
}
