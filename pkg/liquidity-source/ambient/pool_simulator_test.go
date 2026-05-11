package ambient

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	testWETH = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
	testUSDC = "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
)

var testNative = strings.ToLower(valueobject.AddrZero.Hex())

func TestPoolSimulatorCalcAmountOutAndUpdateBalance(t *testing.T) {
	t.Parallel()

	sim := newTestPoolSimulator(t)
	cloned := sim.CloneState().(*PoolSimulator)

	limit := swaplimit.NewInventory(DexType, sim.CalculateLimit())

	require.NotNil(t, limit.GetLimit(testNative))
	require.NotNil(t, limit.GetLimit(testUSDC))
	require.Nil(t, limit.GetLimit(testWETH))

	amountIn := big.NewInt(1_000_000)
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testWETH, Amount: amountIn},
		TokenOut:      testUSDC,
		Limit:         limit,
	})
	require.NoError(t, err)
	require.Positive(t, res.TokenAmountOut.Amount.Sign())
	require.Zero(t, cloned.Info.Reserves[0].Cmp(sim.Info.Reserves[0]), "CalcAmountOut must not mutate state")

	beforeIn := new(big.Int).Set(sim.Info.Reserves[0])
	beforeOut := new(big.Int).Set(sim.Info.Reserves[1])
	quoteBudgetBefore := new(big.Int).Set(limit.GetLimit(testUSDC))
	nativeBudgetBefore := new(big.Int).Set(limit.GetLimit(testNative))

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: testWETH, Amount: amountIn},
		TokenAmountOut: *res.TokenAmountOut,
		Fee:            *res.Fee,
		SwapInfo:       res.SwapInfo,
		SwapLimit:      limit,
	})

	require.Equal(t, new(big.Int).Add(beforeIn, amountIn), sim.Info.Reserves[0])
	require.Equal(t, new(big.Int).Sub(beforeOut, res.TokenAmountOut.Amount), sim.Info.Reserves[1])
	require.Equal(t, new(big.Int).Sub(quoteBudgetBefore, res.TokenAmountOut.Amount), limit.GetLimit(testUSDC))
	require.Equal(t, new(big.Int).Add(nativeBudgetBefore, amountIn), limit.GetLimit(testNative))

	_, _, err = limit.UpdateLimit(testUSDC, testNative, limit.GetLimit(testUSDC), big.NewInt(0))
	require.NoError(t, err)

	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testWETH, Amount: amountIn},
		TokenOut:      testUSDC,
		Limit:         limit,
	})
	require.ErrorIs(t, err, pool.ErrNotEnoughInventory)
}

func TestPoolSimulatorCloneStateIsolatesPairState(t *testing.T) {
	t.Parallel()

	sim := newTestPoolSimulator(t)
	cloned := sim.CloneState().(*PoolSimulator)

	// Production never mutates Curve's big.Ints in place — SweepSwap always
	// reassigns the fields via new(big.Int). Follow that invariant here and
	// verify the clone's state pointer is independent of the original.
	cloned.state.Curve.PriceRoot = new(big.Int).Add(cloned.state.Curve.PriceRoot, big.NewInt(1))
	require.NotEqual(t, 0, cloned.state.Curve.PriceRoot.Cmp(sim.state.Curve.PriceRoot))
}

func newTestPoolSimulator(t *testing.T) *PoolSimulator {
	t.Helper()

	base := valueobject.AddrZero
	quote := common.HexToAddress(testUSDC)
	poolHash := EncodePoolHash(base, quote, 420)

	staticExtra, err := json.Marshal(StaticExtra{
		NativeToken: testWETH,
		PoolIdx:     420,
		SwapDex:     "0xaaaaaaaaa24eeeb8d57d431224f73832bc34f688",
		Base:        base.String(),
		Quote:       testUSDC,
	})
	require.NoError(t, err)

	state := &TrackerExtra{
		Base:     base,
		Quote:    quote,
		PoolIdx:  420,
		PoolHash: poolHash,
		Curve: CurveState{
			PriceRoot:    GetSqrtRatioAtTick(0),
			AmbientSeeds: big.NewInt(1_000_000_000),
			ConcLiq:      big.NewInt(500_000_000),
			SeedDeflator: 0,
			ConcGrowth:   0,
		},
		PoolSpec:    PoolSpec{FeeRate: 2500, ProtocolTake: 0, TickSize: 16},
		PoolParams:  PoolParams{FeeRate: 2500, ProtocolTake: 0, TickSize: 16},
		ActiveTicks: []int32{-256, 256},
		Levels: []TrackedLevel{
			{Tick: -256, Level: BookLevel{BidLots: big.NewInt(0), AskLots: big.NewInt(100)}},
			{Tick: 256, Level: BookLevel{BidLots: big.NewInt(100), AskLots: big.NewInt(0)}},
		},
	}

	extra, err := json.Marshal(Extra{State: state})
	require.NoError(t, err)

	sim, err := NewPoolSimulator(entity.Pool{
		Address:  poolHash.Hex(),
		Exchange: valueobject.ExchangeAmbient,
		Type:     DexType,
		Tokens: []*entity.PoolToken{
			{Address: testWETH, Swappable: true},
			{Address: testUSDC, Swappable: true},
		},
		Reserves:    []string{"1000000000000000000000", "1000000000000"},
		Extra:       string(extra),
		StaticExtra: string(staticExtra),
	})
	require.NoError(t, err)

	return sim
}
