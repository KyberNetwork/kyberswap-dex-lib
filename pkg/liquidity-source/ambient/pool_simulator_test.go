package ambient

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
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

	pair, ok := sim.GetPair(common.HexToAddress(testWETH), common.HexToAddress(testUSDC))
	require.True(t, ok)

	clonedState := cloned.pairInfos[pair].State
	clonedState.Curve.PriceRoot.Add(clonedState.Curve.PriceRoot, big.NewInt(1))

	require.NotEqual(t, 0, clonedState.Curve.PriceRoot.Cmp(sim.pairInfos[pair].State.Curve.PriceRoot))
}

func newTestPoolSimulator(t *testing.T) *PoolSimulator {
	t.Helper()

	staticExtra, err := json.Marshal(StaticExtra{
		NativeTokenAddress: common.HexToAddress(testWETH),
		PoolIdx:            420,
		SwapDex:            common.HexToAddress("0xaaaaaaaaa24eeeb8d57d431224f73832bc34f688"),
	})
	require.NoError(t, err)

	state := &TrackerExtra{
		Base:     NativeTokenPlaceholderAddress,
		Quote:    common.HexToAddress(testUSDC),
		PoolIdx:  420,
		PoolHash: common.HexToHash("0x1"),
		Curve: CurveState{
			PriceRoot:    GetSqrtRatioAtTick(0),
			AmbientSeeds: big.NewInt(1_000_000_000),
			ConcLiq:      big.NewInt(500_000_000),
			SeedDeflator: 0,
			ConcGrowth:   0,
		},
		PoolSpec:       PoolSpec{FeeRate: 2500, ProtocolTake: 0, TickSize: 16},
		TemplateSpec:   PoolSpec{FeeRate: 2500, ProtocolTake: 0, TickSize: 16},
		PoolParams:     PoolParams{FeeRate: 2500, ProtocolTake: 0, TickSize: 16},
		TemplateParams: PoolParams{FeeRate: 2500, ProtocolTake: 0, TickSize: 16},
		ActiveTicks:    []int32{-256, 256},
		Levels: []TrackedLevel{
			{Tick: -256, Level: BookLevel{BidLots: big.NewInt(0), AskLots: big.NewInt(100)}},
			{Tick: 256, Level: BookLevel{BidLots: big.NewInt(100), AskLots: big.NewInt(0)}},
		},
	}

	extra, err := json.Marshal(Extra{
		TokenPairs: map[TokenPair]*TokenPairInfo{
			{Base: NativeTokenPlaceholderAddress, Quote: common.HexToAddress(testUSDC)}: {
				PoolIdx: big.NewInt(420),
				State:   state,
			},
		},
	})
	require.NoError(t, err)

	sim, err := NewPoolSimulator(entity.Pool{
		Address:  "0xaaaaaaaaa24eeeb8d57d431224f73832bc34f688",
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
