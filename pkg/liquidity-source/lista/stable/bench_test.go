package stable

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// go test ./pkg/liquidity-source/lista/stable/ -run '^$' -bench . -benchmem
func BenchmarkLista(b *testing.B) {
	for _, c := range []struct {
		name   string
		pool   entity.Pool
		amount int64
	}{
		{"USD1-USDT 1 USD1", entityUSD1USDT, 1e18},
		{"USDC-USDT 100 USDC", entityUSDCUSDT, 100e6},
	} {
		legacy, err := newLegacyPoolSimulator(c.pool)
		require.NoError(b, err)
		port, err := NewPoolSimulator(c.pool)
		require.NoError(b, err)
		in := poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{Token: port.Info.Tokens[0], Amount: big.NewInt(c.amount)},
			TokenOut:      port.Info.Tokens[1]}
		res, err := port.CalcAmountOut(in)
		require.NoError(b, err)
		update := poolpkg.UpdateBalanceParams{TokenAmountIn: in.TokenAmountIn, TokenAmountOut: *res.TokenAmountOut,
			Fee: *res.Fee}

		for name, sim := range map[string]poolpkg.IPoolSimulator{"legacy": legacy, "uint256": port} {
			b.Run(c.name+"/CalcAmountOut/"+name, func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					if _, err := sim.CalcAmountOut(in); err != nil {
						b.Fatal(err)
					}
				}
			})
			b.Run(c.name+"/CloneState+UpdateBalance/"+name, func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					sim.CloneState().UpdateBalance(update)
				}
			})
		}
	}
}
