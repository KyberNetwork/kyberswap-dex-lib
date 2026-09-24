package prop

import (
	"math/big"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
)

const (
	testWETH = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
	testUSDC = "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
)

func mustSim(t *testing.T, ladders [2][]ladder.Point, bal0, bal1 *big.Int) *PoolSimulator {
	t.Helper()

	extraBytes, err := json.Marshal(Extra{Ladders: ladders})
	require.NoError(t, err)

	staticBytes, err := json.Marshal(StaticExtra{RouterAddress: "0x5cdbe59400cc2efdcc2b54acca4a99fe00dd588c"})
	require.NoError(t, err)

	p := entity.Pool{
		Address:  "kipseli-prop_" + testWETH + "_" + testUSDC,
		Exchange: DexType,
		Type:     DexType,
		Tokens: []*entity.PoolToken{
			{Address: testWETH, Decimals: 18, Swappable: true},
			{Address: testUSDC, Decimals: 6, Swappable: true},
		},
		Reserves:    entity.PoolReserves{bal0.String(), bal1.String()},
		Extra:       string(extraBytes),
		StaticExtra: string(staticBytes),
		Timestamp:   time.Now().Unix(),
	}

	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)
	return sim
}

func calcOut(t *testing.T, sim *PoolSimulator, tokenIn, tokenOut string, amountIn *big.Int) (*big.Int, error) {
	t.Helper()
	limit := swaplimit.NewInventory(DexType, sim.CalculateLimit())
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amountIn},
		TokenOut:      tokenOut,
		Limit:         limit,
	})
	if err != nil {
		return nil, err
	}
	return res.TokenAmountOut.Amount, nil
}

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	ladders := [2][]ladder.Point{
		{{1e18, 3000e6}, {2e18, 5990e6}},
		{{3000e6, 1e18}, {5990e6, 2e18}},
	}
	sim := mustSim(t, ladders, big.NewInt(9e18), big.NewInt(30000e6))

	out, err := calcOut(t, sim, testWETH, testUSDC, big.NewInt(1e18))
	require.NoError(t, err)
	require.Equal(t, big.NewInt(3000e6).String(), out.String())

	// beyond the ladder's last probed point -> ErrAmountInTooLarge from the spline
	_, err = calcOut(t, sim, testWETH, testUSDC, big.NewInt(3e18))
	require.Error(t, err)
}

func TestPoolSimulator_UpdateBalanceConsumesLadder(t *testing.T) {
	ladders := [2][]ladder.Point{
		{{1e18, 3000e6}, {2e18, 5990e6}},
		{{3000e6, 1e18}, {5990e6, 2e18}},
	}
	sim := mustSim(t, ladders, big.NewInt(9e18), big.NewInt(30000e6))

	limit := swaplimit.NewInventory(DexType, sim.CalculateLimit())
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testWETH, Amount: big.NewInt(1e18)},
		TokenOut:      testUSDC,
		Limit:         limit,
	})
	require.NoError(t, err)

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: testWETH, Amount: big.NewInt(1e18)},
		TokenAmountOut: *res.TokenAmountOut,
		SwapLimit:      limit,
	})

	// a second 1e18 swap now needs the ladder's second knot (2e18 total consumed)
	out2, err := calcOut(t, sim, testWETH, testUSDC, big.NewInt(1e18))
	require.NoError(t, err)
	require.Equal(t, new(big.Int).Sub(big.NewInt(5990e6), res.TokenAmountOut.Amount).String(), out2.String())
}

func TestPoolSimulator_CloneStateIsolatesReserves(t *testing.T) {
	ladders := [2][]ladder.Point{
		{{1e18, 3000e6}},
		{{3000e6, 1e18}},
	}
	sim := mustSim(t, ladders, big.NewInt(9e18), big.NewInt(30000e6))

	cloned := sim.CloneState().(*PoolSimulator)
	limit := swaplimit.NewInventory(DexType, sim.CalculateLimit())
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testWETH, Amount: big.NewInt(1e18)},
		TokenOut:      testUSDC,
		Limit:         limit,
	})
	require.NoError(t, err)
	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: testWETH, Amount: big.NewInt(1e18)},
		TokenAmountOut: *res.TokenAmountOut,
		SwapLimit:      limit,
	})

	require.NotEqual(t, sim.GetReserves()[0].String(), cloned.GetReserves()[0].String())
}
