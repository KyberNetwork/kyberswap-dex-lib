package ambient

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	testWETH = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
	testUSDC = "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
)

func TestPoolSimulatorCalcAmountOutAndUpdateBalance(t *testing.T) {
	t.Parallel()

	sim := newTestPoolSimulator(t)
	cloned := sim.CloneState().(*PoolSimulator)

	amountIn := big.NewInt(1_000_000)
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testWETH, Amount: amountIn},
		TokenOut:      testUSDC,
	})
	require.NoError(t, err)
	require.Positive(t, res.TokenAmountOut.Amount.Sign())
	require.Zero(t, cloned.Info.Reserves[0].Cmp(sim.Info.Reserves[0]), "CalcAmountOut must not mutate state")

	beforeIn := new(big.Int).Set(sim.Info.Reserves[0])
	beforeOut := new(big.Int).Set(sim.Info.Reserves[1])

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: testWETH, Amount: amountIn},
		TokenAmountOut: *res.TokenAmountOut,
		Fee:            *res.Fee,
		SwapInfo:       res.SwapInfo,
	})

	require.Equal(t, new(big.Int).Add(beforeIn, amountIn), sim.Info.Reserves[0])
	require.Equal(t, new(big.Int).Sub(beforeOut, res.TokenAmountOut.Amount), sim.Info.Reserves[1])
}

func TestPoolSimulatorCloneStateDeepCopiesPairState(t *testing.T) {
	t.Parallel()

	sim := newTestPoolSimulator(t)
	cloned := sim.CloneState().(*PoolSimulator)

	cloned.state.Curve.PriceRoot.Add(cloned.state.Curve.PriceRoot, big.NewInt(1))
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
		Base:        base.Hex(),
		Quote:       quote.Hex(),
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
		Exchange: DexType,
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
