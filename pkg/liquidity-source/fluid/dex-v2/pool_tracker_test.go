package dexv2

import (
	"context"
	"math/big"
	"testing"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
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
