package volatile

import (
	"math/big"
	"os"
	"testing"

	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// newBenchSim builds the real ETH/USDC pool (prod ticks) with the given hook state.
func newBenchSim(tb testing.TB, hookExtra string) *uniswapv4.PoolSimulator {
	tb.Helper()
	raw, err := os.ReadFile("testdata/pool.json")
	if err != nil {
		tb.Fatal(err)
	}
	var ep entity.Pool
	if err := json.Unmarshal(raw, &ep); err != nil {
		tb.Fatal(err)
	}
	var extra map[string]json.RawMessage
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		tb.Fatal(err)
	}
	extra["hX"] = json.RawMessage(hookExtra)
	b, _ := json.Marshal(extra)
	ep.Extra = string(b)
	sim, err := uniswapv4.NewPoolSimulator(ep, valueobject.ChainIDEthereum)
	if err != nil {
		tb.Fatal(err)
	}
	return sim
}

func BenchmarkCalcAmountOut(b *testing.B) {
	sim := newBenchSim(b, benchHookExtra)
	const weth, usdc = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
	for _, bc := range []struct {
		name, in, out string
		amt           *big.Int
	}{{"sell_eth", weth, usdc, big.NewInt(1e18)}, {"buy_eth", usdc, weth, big.NewInt(2_000_000_000)}} {
		params := pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: bc.in, Amount: bc.amt}, TokenOut: bc.out}
		b.Run(bc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := sim.CalcAmountOut(params); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
