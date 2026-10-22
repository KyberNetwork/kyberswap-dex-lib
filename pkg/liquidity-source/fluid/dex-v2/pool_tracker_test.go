package dexv2

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/fluid/dex-v2/abis"
	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const testPoolAddress = "0x8d1b5f8da63fa29b191672231d3845740a11fcbef6c76e077cfffe56cc27c707_3"

// readsAround are Polygon's USDC/USDT D3 pool read just before a swap at block 93210176 and well
// after it: the swap moved the token reserves and both exchange prices.
var readsAround = [2]string{
	`{"liquidity":198354195383694,"sqrtPriceX96":79271759844157699018832478208,"tick":11,"dexVariables2":9134385233318143238773084727662829503992807030784,"token0ExchangePricesAndConfig":47974002575691986918833298772910046115536681496709052236044,"token1ExchangePricesAndConfig":48090179374933996073259782694874751141862369554426086556233,"tokenReserves":631038166607185477792740625937576002715287841766}`,
	`{"liquidity":198354195383694,"sqrtPriceX96":79271747915661983442773475328,"tick":11,"dexVariables2":9134385233318143238773084727662829503992807030784,"token0ExchangePricesAndConfig":48203429341036302796212388712214021330328421008310528377269,"token1ExchangePricesAndConfig":48459812694683311653602790311961131657315967176295496286643,"tokenReserves":620877001994489039674968529515922728220162497473}`,
}

func mustExtra(t *testing.T, raw string) Extra {
	t.Helper()
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(raw), &extra))
	return extra
}

func testTicks() []Tick {
	return []Tick{
		{Index: -10, LiquidityGross: big.NewInt(1000), LiquidityNet: big.NewInt(1000)},
		{Index: 20, LiquidityGross: big.NewInt(1000), LiquidityNet: big.NewInt(-1000)},
	}
}

// triggerLog is the empty log the caller sends to update a pool on an interval, with no event.
func triggerLog(block uint64) []ethtypes.Log {
	return []ethtypes.Log{{BlockNumber: block}}
}

// TestUpdateStateTakesEveryFieldFromTheRead: an update writes the fee bits and the token reserves
// of its own read, not the bootstrap's, as the simulator prices swaps and checks reserves with them.
func TestUpdateStateTakesEveryFieldFromTheRead(t *testing.T) {
	before, after := mustExtra(t, readsAround[0]), mustExtra(t, readsAround[1])
	// A fee change, which the swap alone did not make.
	before.DexVariables2 = new(big.Int).Xor(after.DexVariables2, new(big.Int).Lsh(big.NewInt(100^500),
		BITS_DEX_V2_VARIABLES2_LP_FEE))
	before.Ticks = testTicks()

	p, err := withState(entity.Pool{Address: testPoolAddress, BlockNumber: 100}, before)
	require.NoError(t, err)
	staleReserves := p.Reserves

	tracker := &PoolTracker{config: &liveConfig}
	tracker.readState = func(context.Context, *entity.Pool, uint64) (Extra, error) {
		return mustExtra(t, readsAround[1]), nil
	}
	p, err = tracker.GetNewPoolState(t.Context(), p, poolpkg.GetNewPoolStateParams{Logs: triggerLog(101)})
	require.NoError(t, err)

	got := mustExtra(t, p.Extra)
	assert.Zero(t, got.DexVariables2.Cmp(after.DexVariables2), "dexVariables2")
	assert.Zero(t, got.TokenReserves.Cmp(after.TokenReserves), "tokenReserves")
	assert.Zero(t, got.Token0ExchangePricesAndConfig.Cmp(after.Token0ExchangePricesAndConfig))
	assert.Equal(t, testTicks(), got.Ticks, "the ticks are the pool's, not the read's")

	want, err := withState(entity.Pool{Address: testPoolAddress}, after)
	require.NoError(t, err)
	assert.Equal(t, want.Reserves, p.Reserves)
	assert.NotEqual(t, staleReserves, p.Reserves, "the swap moved the reserves")
}

// TestUpdateStateOfAFactoryPool: a pool the factory built from its LogInitialize, whose Extra is
// "{}", gets its fee bits and reserves on its first update instead of failing on their absence.
func TestUpdateStateOfAFactoryPool(t *testing.T) {
	tracker := &PoolTracker{config: &liveConfig}
	tracker.readState = func(context.Context, *entity.Pool, uint64) (Extra, error) {
		return mustExtra(t, readsAround[1]), nil
	}
	p := entity.Pool{Address: testPoolAddress, Extra: "{}", Reserves: entity.PoolReserves{"0", "0"}}
	p, err := tracker.GetNewPoolState(t.Context(), p, poolpkg.GetNewPoolStateParams{Logs: triggerLog(101)})
	require.NoError(t, err)

	got := mustExtra(t, p.Extra)
	require.NotNil(t, got.DexVariables2)
	assert.NotEqual(t, entity.PoolReserves{"0", "0"}, p.Reserves)
	assert.Positive(t, bignumber.NewBig10(p.Reserves[0]).Sign())
}

// depositLog is a LogDeposit adding liquidity to [lower, upper] of the test pool at block, with its
// dexType, dexId and user in the topics, as the dex emits it.
func depositLog(t *testing.T, block uint64, lower, upper int64, liquidity int64) ethtypes.Log {
	t.Helper()
	ev := abis.DexV2ABI.Events["LogDeposit"]
	dexID, dexType := parseFluidDexV2PoolAddress(testPoolAddress)
	data, err := ev.Inputs.NonIndexed().Pack(big.NewInt(lower), big.NewInt(upper), [32]byte{}, big.NewInt(1),
		big.NewInt(1), big.NewInt(0), big.NewInt(0), big.NewInt(liquidity))
	require.NoError(t, err)
	topics := []common.Hash{ev.ID, common.BigToHash(big.NewInt(int64(dexType))), common.HexToHash(dexID),
		common.BytesToHash(common.Address{1}.Bytes())}
	return ethtypes.Log{Address: common.HexToAddress(liveConfig.Dex), Topics: topics, Data: data, BlockNumber: block}
}

// pinnedTicks stands in for the chain's tick reads, answering testTicks at any block and recording
// the reads.
type pinnedTicks struct{ reads []uint64 }

func (c *pinnedTicks) read(_ context.Context, _ string, indexes []int, block uint64) ([]tickspkg.Tick, error) {
	c.reads = append(c.reads, block)
	byIndex := lo.KeyBy(testTicks(), func(tk Tick) int { return tk.Index })
	return lo.Map(indexes, func(i int, _ int) tickspkg.Tick {
		tk := byIndex[i]
		return tickspkg.Tick{TickIdx: i, LiquidityGross: tk.LiquidityGross, LiquidityNet: tk.LiquidityNet}
	}), nil
}

// TestLateLogsAreReadWhole: a log at or below the pool's block, which its ticks may hold already,
// has the ticks it touches read whole at the pool's block rather than its delta added again; a log
// above it is added.
func TestLateLogsAreReadWhole(t *testing.T) {
	// The pool at block 100 holds testTicks, a deposit of 1000 to [-10, 20] made at block 100.
	start := mustExtra(t, readsAround[1])
	start.Ticks = testTicks()
	pool, err := withState(entity.Pool{Address: testPoolAddress}, start)
	require.NoError(t, err)
	pool.BlockNumber = 100

	for _, tc := range []struct {
		name      string
		block     uint64
		wantGross int64
		wantReads []uint64
	}{
		{"replayed", 100, 1000, []uint64{100}},
		{"late", 90, 1000, []uint64{100}},
		{"live", 101, 2000, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			chain := &pinnedTicks{}
			tracker := &PoolTracker{config: &liveConfig, readTicks: chain.read}
			tracker.readState = func(context.Context, *entity.Pool, uint64) (Extra, error) {
				return mustExtra(t, readsAround[1]), nil
			}
			p, err := tracker.GetNewPoolState(t.Context(), pool,
				poolpkg.GetNewPoolStateParams{Logs: []ethtypes.Log{depositLog(t, tc.block, -10, 20, 1000)}})
			require.NoError(t, err)

			got := mustExtra(t, p.Extra)
			require.Len(t, got.Ticks, 2)
			assert.Equal(t, tc.wantGross, got.Ticks[0].LiquidityGross.Int64())
			assert.Equal(t, tc.wantReads, chain.reads)
		})
	}
}

// TestPoolWithoutStateIsNotSimulated: a pool the factory built, or one reset for the lens to read
// whole, is refused by the simulator rather than read as zeros.
func TestPoolWithoutStateIsNotSimulated(t *testing.T) {
	start := mustExtra(t, readsAround[1])
	start.Ticks = testTicks()
	p, err := withState(entity.Pool{Address: testPoolAddress, StaticExtra: `{"tickSpacing":1}`,
		Tokens: []*entity.PoolToken{{Address: "0x1"}, {Address: "0x2"}}}, start)
	require.NoError(t, err)
	reset, ok := (&PoolTracker{config: &Config{UseLens: true}}).ResetPoolState(p, 7)
	require.True(t, ok)

	for _, extra := range []string{"{}", reset.Extra} {
		p.Extra = extra
		_, err := NewPoolSimulator(p, 137)
		assert.ErrorIs(t, err, ErrV3TicksEmpty, extra)
	}
}

// TestResetPoolState: a reset drops the state logs built and keeps the rest, for the lens to read
// the pool whole at block 7 or later; without the lens nothing can, and the pool is kept.
func TestResetPoolState(t *testing.T) {
	start := mustExtra(t, readsAround[1])
	start.Ticks = testTicks()
	start.LensBlock = 5
	p, err := withState(entity.Pool{Address: testPoolAddress}, start)
	require.NoError(t, err)

	kept, ok := (&PoolTracker{config: &Config{}}).ResetPoolState(p, 7)
	assert.False(t, ok)
	assert.Equal(t, p, kept)

	tracker := NewPoolTracker(&Config{UseLens: true}, nil, nil)
	reset, ok := tracker.ResetPoolState(p, 7)
	require.True(t, ok)
	assert.False(t, tickspkg.HasState(reset))
	assert.Equal(t, uint64(7), tickspkg.LensDue(reset))
	got := mustExtra(t, reset.Extra)
	assert.Empty(t, got.Ticks)
	assert.Zero(t, got.LensBlock)
	assert.Zero(t, got.TokenReserves.Cmp(start.TokenReserves), "the rest of the state is kept")
	assert.NotZero(t, tracker.NextUpdateAt(reset), "a pool without state is updated again")
	assert.Zero(t, tracker.NextUpdateAt(p))
}
