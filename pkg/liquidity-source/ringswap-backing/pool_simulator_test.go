package ringswapbacking

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
)

const (
	testToken0   = "0x0000000000000000000000000000000000000010"
	testToken1   = "0x0000000000000000000000000000000000000020"
	testWrapper0 = "0x0000000000000000000000000000000000000030"
	testWrapper1 = "0x0000000000000000000000000000000000000040"
	testRouter   = "0x0000000000000000000000000000000000000050"
	testPair     = "0x0000000000000000000000000000000000000060"

	testNoRecallGas0 int64 = 245_000
	testNoRecallGas1 int64 = 250_000
	testRecallGas0   int64 = 410_000
	testRecallGas1   int64 = 390_000
)

func TestPoolSimulatorUsesDirectHotBackingPathWithoutRecall(t *testing.T) {
	for _, direction := range []struct {
		in          string
		out         string
		expectedGas int64
	}{
		{testToken0, testToken1, testNoRecallGas1},
		{testToken1, testToken0, testNoRecallGas0},
	} {
		simulator := newTestSimulator(t)
		result, err := simulator.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: direction.in, Amount: units(100)},
			TokenOut:      direction.out,
			Limit:         newTestLimit(simulator),
		})
		require.NoError(t, err)
		require.Equal(t, direction.expectedGas, result.Gas)
		require.False(t, result.SwapInfo.(SwapInfo).UseRecall)
	}
}

func TestPoolSimulatorQuotesOriginalPairDepthForRecallInBothDirections(t *testing.T) {
	for _, direction := range []struct {
		in          string
		out         string
		expectedGas int64
	}{
		{testToken0, testToken1, testRecallGas1},
		{testToken1, testToken0, testRecallGas0},
	} {
		simulator := newTestSimulator(t)
		amountIn := units(20_000)
		result, err := simulator.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: direction.in, Amount: amountIn},
			TokenOut:      direction.out,
			Limit:         newTestLimit(simulator),
		})
		require.NoError(t, err)
		require.Equal(t, direction.expectedGas, result.Gas)
		require.Equal(
			t,
			amountOut(amountIn, units(150_000), units(150_000)),
			result.TokenAmountOut.Amount,
			"recall capacity must not be added to AMM reserves",
		)
		require.True(t, result.SwapInfo.(SwapInfo).UseRecall)
	}
}

func TestPoolSimulatorRejectsOutputAboveSharedDeliverableBacking(t *testing.T) {
	simulator := newTestSimulator(t)
	limit := newTestLimit(simulator)
	limit = swaplimit.NewInventory(DexType, map[string]*big.Int{
		testWrapper0: units(100_000),
		testWrapper1: units(15_000),
	})
	_, err := simulator.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testToken0, Amount: units(20_000)},
		TokenOut:      testToken1,
		Limit:         limit,
	})
	require.ErrorIs(t, err, ErrInsufficientBacking)
}

func TestPoolSimulatorRequiresSharedBackingLimit(t *testing.T) {
	simulator := newTestSimulator(t)
	_, err := simulator.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testToken0, Amount: units(20_000)},
		TokenOut:      testToken1,
	})
	require.ErrorIs(t, err, ErrNoSwapLimit)
}

func TestPoolSimulatorUpdateConsumesSourceAndSharedInventory(t *testing.T) {
	simulator := newTestSimulator(t)
	clone := simulator.CloneState().(*PoolSimulator)
	limit := newTestLimit(simulator)
	amountIn := units(20_000)
	result, err := simulator.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testToken0, Amount: amountIn},
		TokenOut:      testToken1,
		Limit:         limit,
	})
	require.NoError(t, err)
	simulator.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: testToken0, Amount: amountIn},
		TokenAmountOut: *result.TokenAmountOut,
		SwapInfo:       result.SwapInfo,
		SwapLimit:      limit,
	})

	require.Equal(
		t,
		new(big.Int).Sub(units(100_000), result.TokenAmountOut.Amount),
		limit.GetLimit(testWrapper1),
	)
	require.Equal(t, units(120_000), limit.GetLimit(testWrapper0))
	_, err = simulator.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testToken0, Amount: amountIn},
		TokenOut:      testToken1,
		Limit:         limit,
	})
	require.ErrorIs(t, err, ErrSourceAlreadyUsed)
	_, err = clone.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testToken0, Amount: amountIn},
		TokenOut:      testToken1,
		Limit:         newTestLimit(clone),
	})
	require.NoError(t, err)
}

func TestPoolSimulatorMetaExposesOriginalPairAndBackingRouter(t *testing.T) {
	simulator := newTestSimulator(t)
	require.Equal(t, testPair, simulator.GetAddress())
	meta := simulator.GetMetaInfo(testToken0, testToken1).(PoolMeta)
	require.Equal(t, testRouter, meta.ApprovalAddress)
	require.Equal(t, testPair, meta.UnderlyingPair)
	require.True(t, meta.SingleUse)
	require.True(t, meta.ReplacesOrdinaryPair)
}

func TestPoolSimulatorRejectsMalformedState(t *testing.T) {
	_, err := NewPoolSimulator(newTestPoolEntity(t, testRouter))
	require.ErrorIs(t, err, ErrInvalidState)

	entityPool := newTestPoolEntity(t, testPair)
	entityPool.Extra = "{}"
	_, err = NewPoolSimulator(entityPool)
	require.ErrorIs(t, err, ErrInvalidState)
}

func TestPoolSimulatorRejectsNilInputAmount(t *testing.T) {
	simulator := newTestSimulator(t)
	_, err := simulator.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testToken0},
		TokenOut:      testToken1,
		Limit:         newTestLimit(simulator),
	})
	require.ErrorIs(t, err, uniswapv2.ErrInvalidAmountIn)
}

func TestPoolSimulatorMatchesFixedMainnetPairReserveFixture(t *testing.T) {
	// Ring fwWETH/fwUSDT reserves at mainnet block 25,503,661. Expected outputs were also
	// asserted by FewBackingAwareV2RouterForkTest against the Solidity route.
	tests := []struct {
		name     string
		tokenIn  string
		tokenOut string
		amountIn *big.Int
		expected *big.Int
	}{
		{"weth-0.1", testToken0, testToken1, big.NewInt(100_000_000_000_000_000), big.NewInt(178_829_383)},
		{"weth-5", testToken0, testToken1, big.NewInt(5_000_000_000_000_000_000), big.NewInt(8_938_332_759)},
		{"usdt-5000", testToken1, testToken0, big.NewInt(5_000_000_000), big.NewInt(2_778_635_928_725_029_183)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			simulator := newForkFixtureSimulator(t)
			result, err := simulator.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: tt.tokenIn, Amount: tt.amountIn},
				TokenOut:      tt.tokenOut,
				Limit:         newTestLimit(simulator),
			})
			require.NoError(t, err)
			require.Equal(t, tt.expected, result.TokenAmountOut.Amount)
		})
	}
}

func TestConfigFailsClosedOnMissingGasOrInvalidRouter(t *testing.T) {
	config := &Config{DexID: DexType, Routers: []RouterConfig{{Address: testRouter}}}
	require.ErrorIs(t, config.validate(), ErrInvalidConfig)
	config.Routers[0] = RouterConfig{
		Address: "not-an-address", ReplaceOrdinaryPair: true, NoRecallGasToken0: 1,
		NoRecallGasToken1: 1, RecallGasToken0: 1, RecallGasToken1: 1,
	}
	require.ErrorIs(t, config.validate(), ErrInvalidConfig)
	config.Routers[0].Address = testRouter
	require.NoError(t, config.validate())
	config.Routers = append(config.Routers, config.Routers[0])
	require.ErrorIs(t, config.validate(), ErrInvalidConfig)
	config.Routers = config.Routers[:1]
	config.Routers[0].RecallGasToken0 = config.Routers[0].NoRecallGasToken0 - 1
	require.ErrorIs(t, config.validate(), ErrInvalidConfig)
}

func TestKnownSourceMetadataRequiresOneUniquePairPerRouter(t *testing.T) {
	routers, pairs, err := knownSourceSet(PoolsListUpdaterMetadata{
		KnownRouters: []string{testRouter},
		KnownPairs:   []string{testPair},
	})
	require.NoError(t, err)
	require.Contains(t, routers, testRouter)
	require.Contains(t, pairs, testPair)

	_, _, err = knownSourceSet(PoolsListUpdaterMetadata{KnownRouters: []string{testRouter}})
	require.ErrorIs(t, err, ErrInvalidMetadata)
	_, _, err = knownSourceSet(PoolsListUpdaterMetadata{
		KnownRouters: []string{testRouter, "0x0000000000000000000000000000000000000051"},
		KnownPairs:   []string{testPair, testPair},
	})
	require.ErrorIs(t, err, ErrInvalidMetadata)
}

func newTestSimulator(t *testing.T) *PoolSimulator {
	t.Helper()
	simulator, err := NewPoolSimulator(newTestPoolEntity(t, testPair))
	require.NoError(t, err)
	return simulator
}

func newTestPoolEntity(t *testing.T, address string) entity.Pool {
	t.Helper()
	extra, err := json.Marshal(Extra{
		WrapperBuffer0:  units(10_000),
		WrapperBuffer1:  units(10_000),
		RecallCapacity0: units(90_000),
		RecallCapacity1: units(90_000),
	})
	require.NoError(t, err)
	staticExtra, err := json.Marshal(StaticExtra{
		RouterAddress:       testRouter,
		PairAddress:         testPair,
		Wrapper0:            testWrapper0,
		Wrapper1:            testWrapper1,
		ReplaceOrdinaryPair: true,
		NoRecallGasToken0:   testNoRecallGas0,
		NoRecallGasToken1:   testNoRecallGas1,
		RecallGasToken0:     testRecallGas0,
		RecallGasToken1:     testRecallGas1,
	})
	require.NoError(t, err)
	return entity.Pool{
		Address:     address,
		Exchange:    DexType,
		Type:        DexType,
		Reserves:    entity.PoolReserves{units(150_000).String(), units(150_000).String()},
		Tokens:      []*entity.PoolToken{{Address: testToken0}, {Address: testToken1}},
		Extra:       string(extra),
		StaticExtra: string(staticExtra),
	}
}

func newForkFixtureSimulator(t *testing.T) *PoolSimulator {
	t.Helper()
	entityPool := newTestPoolEntity(t, testPair)
	entityPool.Reserves = entity.PoolReserves{"13922377412137157664837", "24972397132732"}
	extra, err := json.Marshal(Extra{
		WrapperBuffer0:  big.NewInt(0),
		WrapperBuffer1:  big.NewInt(0),
		RecallCapacity0: new(big.Int).SetUint64(^uint64(0)),
		RecallCapacity1: new(big.Int).SetUint64(^uint64(0)),
	})
	require.NoError(t, err)
	entityPool.Extra = string(extra)
	simulator, err := NewPoolSimulator(entityPool)
	require.NoError(t, err)
	return simulator
}

func newTestLimit(simulator *PoolSimulator) *swaplimit.Inventory {
	return swaplimit.NewInventory(DexType, simulator.CalculateLimit())
}

func units(amount int64) *big.Int {
	return new(big.Int).Mul(big.NewInt(amount), big.NewInt(1_000_000))
}

func amountOut(amountIn, reserveIn, reserveOut *big.Int) *big.Int {
	amountInWithFee := new(big.Int).Mul(amountIn, big.NewInt(997))
	numerator := new(big.Int).Mul(amountInWithFee, reserveOut)
	denominator := new(big.Int).Add(new(big.Int).Mul(reserveIn, big.NewInt(1_000)), amountInWithFee)
	return numerator.Div(numerator, denominator)
}
