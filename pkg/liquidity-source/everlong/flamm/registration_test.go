package everlongflamm_test

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	everlongflamm "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/everlong/flamm"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpack"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// pool-service builds the tracker from the registry and refreshes on a state it overrides through
// GetNewPoolStateWithOverrides, which the registered tracker serves, its config's sampler keys read.
func TestTrackerFactoryServesOverridesAndASchedule(t *testing.T) {
	props := pooltrack.Properties{"sampleGas": 30_000_000, "disableSampler": true}
	var cfg everlongflamm.Config
	require.NoError(t, pool.PropertiesToStruct(props, &cfg))
	require.Equal(t, uint64(30_000_000), cfg.SampleGas)
	require.True(t, cfg.DisableSampler)

	factory := pooltrack.Factory(everlongflamm.DexType)
	require.NotNil(t, factory)
	tracker, err := factory(everlongflamm.DexType, pooltrack.FactoryParams{Properties: props})
	require.NoError(t, err)
	require.Implements(t, (*pool.IPoolTrackerWithOverrides)(nil), tracker)
	require.Implements(t, (*pool.IPoolTrackerWithSchedule)(nil), tracker)
}

func TestFactoryAndSerialization(t *testing.T) {
	require.Equal(t, valueobject.ExchangeEverlongFlamm, everlongflamm.DexType)
	factory := pool.Factory(everlongflamm.DexType)
	require.NotNil(t, factory)
	p := entity.Pool{
		Address:  "0x1111111111111111111111111111111111111111",
		Exchange: everlongflamm.DexType, Type: everlongflamm.DexType,
		Timestamp: time.Now().Unix(), BlockNumber: 123,
		Tokens: []*entity.PoolToken{
			{Address: "0x2222222222222222222222222222222222222222", Swappable: true},
			{Address: "0x3333333333333333333333333333333333333333", Swappable: true},
		},
		Reserves: entity.PoolReserves{"1000", "1000"},
		// the sale's custody holds 100 of its output: a fill paying more is charged a venue's path, and a fill after
		// another on the pool its second-fill charge
		Extra: `{"l":[{"i":0,"j":1,"p":[["100","200"],["200","390"],["400","770"]],` +
			`"cs":[{"p":0,"g":500000},{"p":1,"g":700000}],"sc":300000,"ho":"100"},` +
			`{"i":1,"j":0,"p":[["200","100"],["400","195"]],"c":true,"cs":[{"p":0,"g":700000}],"sc":300000}],"h":1}`,
	}
	sim, err := factory(pool.FactoryParams{EntityPool: p, Opts: pool.FactoryOpts{StaleCheck: true}})
	require.NoError(t, err)
	q := pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: p.Tokens[0].Address, Amount: big.NewInt(125)},
		TokenOut:      p.Tokens[1].Address,
	}
	first, err := sim.CalcAmountOut(q)
	require.NoError(t, err)
	require.Equal(t, int64(700_000), first.Gas, "past custody: the venue's path")
	encoded, err := msgpack.EncodePoolSimulatorsMap(map[string]pool.IPoolSimulator{p.Address: sim})
	require.NoError(t, err)
	restored, err := msgpack.DecodePoolSimulatorsMap(encoded)
	require.NoError(t, err)
	fresh, err := restored[p.Address].CalcAmountOut(q)
	require.NoError(t, err)
	require.Equal(t, first, fresh, "the charge's reads survive the round trip")
	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn: q.TokenAmountIn, TokenAmountOut: *first.TokenAmountOut, Fee: *first.Fee,
		SwapInfo: first.SwapInfo,
	})
	_, err = sim.CalcAmountOut(q)
	require.ErrorIs(t, err, ladder.ErrNoQuote, "a direction takes one fill per route")
	// the other direction is quoted, at its second-fill charge
	back := pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: p.Tokens[1].Address, Amount: big.NewInt(200)},
		TokenOut:      p.Tokens[0].Address,
	}
	expected, err := sim.CalcAmountOut(back)
	require.NoError(t, err)
	require.Equal(t, int64(300_000), expected.Gas, "after a fill on the pool: the second fill")
	encoded, err = msgpack.EncodePoolSimulatorsMap(map[string]pool.IPoolSimulator{p.Address: sim})
	require.NoError(t, err)
	restored, err = msgpack.DecodePoolSimulatorsMap(encoded)
	require.NoError(t, err)
	decoded := restored[p.Address]
	require.IsType(t, &everlongflamm.PoolSimulator{}, decoded)
	actual, err := decoded.CalcAmountOut(back)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
	_, err = decoded.CalcAmountOut(q)
	require.ErrorIs(t, err, ladder.ErrNoQuote, "the one fill of the direction, after the round trip")
	require.Equal(t, sim.GetReserves(), decoded.GetReserves())
	require.Equal(t, everlongflamm.PoolMeta{ApprovalAddress: p.Address, BlockNumber: 123},
		decoded.GetMetaInfo(q.TokenAmountIn.Token, q.TokenOut))

	// a direction with a flat span takes one fill, and still does once its simulator has been round-tripped
	p.Extra = `{"l":[{"i":0,"j":1,"p":[["100","200"],["200","390"],["400","770"]],"f":4,"cs":[{"p":1,"g":700000}],"sc":300000}],"h":1}`
	sim, err = factory(pool.FactoryParams{EntityPool: p, Opts: pool.FactoryOpts{StaleCheck: true}})
	require.NoError(t, err)
	first, err = sim.CalcAmountOut(q)
	require.NoError(t, err)
	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn: q.TokenAmountIn, TokenAmountOut: *first.TokenAmountOut, Fee: *first.Fee,
		SwapInfo: first.SwapInfo,
	})
	_, err = sim.CalcAmountOut(q)
	require.ErrorIs(t, err, ladder.ErrNoQuote)
	encoded, err = msgpack.EncodePoolSimulatorsMap(map[string]pool.IPoolSimulator{p.Address: sim})
	require.NoError(t, err)
	restored, err = msgpack.DecodePoolSimulatorsMap(encoded)
	require.NoError(t, err)
	_, err = restored[p.Address].CalcAmountOut(q)
	require.ErrorIs(t, err, ladder.ErrNoQuote, "the second fill, after the round trip")

	// and a fresh one round-trips its flat span: the first fill quotes the point below it
	sim, err = factory(pool.FactoryParams{EntityPool: p, Opts: pool.FactoryOpts{StaleCheck: true}})
	require.NoError(t, err)
	encoded, err = msgpack.EncodePoolSimulatorsMap(map[string]pool.IPoolSimulator{p.Address: sim})
	require.NoError(t, err)
	restored, err = msgpack.DecodePoolSimulatorsMap(encoded)
	require.NoError(t, err)
	q.TokenAmountIn.Amount = big.NewInt(300)
	res, err := restored[p.Address].CalcAmountOut(q)
	require.NoError(t, err)
	require.Equal(t, "390", res.TokenAmountOut.Amount.String())

	// a sale past what the pool frees, the output owing nothing and the input's collateral posted, is charged an
	// owing borrow, and still is once its simulator has been round-tripped
	p.Extra = `{"l":[{"i":0,"j":1,"p":[["100","200"],["200","390"],["400","770"]],` +
		`"cs":[{"p":3,"g":900000},{"p":5,"g":800000}],"sc":300000,"fr":"100","oo":1,"pi":true}],"h":1}`
	sim, err = factory(pool.FactoryParams{EntityPool: p, Opts: pool.FactoryOpts{StaleCheck: true}})
	require.NoError(t, err)
	q.TokenAmountIn.Amount = big.NewInt(125)
	first, err = sim.CalcAmountOut(q)
	require.NoError(t, err)
	require.Equal(t, int64(800_000), first.Gas, "collateral posted ahead: an owing borrow")
	encoded, err = msgpack.EncodePoolSimulatorsMap(map[string]pool.IPoolSimulator{p.Address: sim})
	require.NoError(t, err)
	restored, err = msgpack.DecodePoolSimulatorsMap(encoded)
	require.NoError(t, err)
	fresh, err = restored[p.Address].CalcAmountOut(q)
	require.NoError(t, err)
	require.Equal(t, first, fresh, "the posted read survives the round trip")

	// a pool's first fill after another pool of the source filled in the route, as the route's swap limit says, is
	// charged its path's cell after another pool, and still is once its simulator has been round-tripped; with no
	// limit, or one no fill was applied with, its cell
	p.Extra = `{"l":[{"i":0,"j":1,"p":[["100","200"],["200","390"],["400","770"]],` +
		`"cs":[{"p":0,"g":500000},{"p":1,"g":700000}],"sc":300000,"ac":[{"p":0,"g":420000},{"p":1,"g":610000}],` +
		`"ho":"100"},{"i":1,"j":0,"p":[["200","100"],["400","195"]],"cs":[{"p":0,"g":700000}],"sc":300000}],"h":1}`
	sim, err = factory(pool.FactoryParams{EntityPool: p, Opts: pool.FactoryOpts{StaleCheck: true}})
	require.NoError(t, err)
	other := p
	other.Address = "0x4444444444444444444444444444444444444444"
	otherSim, err := factory(pool.FactoryParams{EntityPool: other, Opts: pool.FactoryOpts{StaleCheck: true}})
	require.NoError(t, err)
	limits := map[string]*big.Int{}
	pool.MergeSwapLimits(limits, sim)
	pool.MergeSwapLimits(limits, otherSim)
	require.Len(t, limits, 1)
	limit := swaplimit.NewInventory(everlongflamm.DexType, limits)
	first, err = sim.CalcAmountOut(q)
	require.NoError(t, err)
	require.Equal(t, int64(700_000), first.Gas, "no limit: the venue's path")
	q.Limit = limit
	unmarked, err := sim.CalcAmountOut(q)
	require.NoError(t, err)
	require.Equal(t, first, unmarked, "no fill applied with the limit")
	otherFill, err := otherSim.CalcAmountOut(q)
	require.NoError(t, err)
	otherSim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn: q.TokenAmountIn, TokenAmountOut: *otherFill.TokenAmountOut, Fee: *otherFill.Fee,
		SwapInfo: otherFill.SwapInfo, SwapLimit: limit,
	})
	after, err := sim.CalcAmountOut(q)
	require.NoError(t, err)
	require.Equal(t, int64(610_000), after.Gas, "after another pool: the path's cell after another pool")
	require.Equal(t, first.TokenAmountOut, after.TokenAmountOut)
	encoded, err = msgpack.EncodePoolSimulatorsMap(map[string]pool.IPoolSimulator{p.Address: sim,
		other.Address: otherSim})
	require.NoError(t, err)
	restored, err = msgpack.DecodePoolSimulatorsMap(encoded)
	require.NoError(t, err)
	fresh, err = restored[p.Address].CalcAmountOut(q)
	require.NoError(t, err)
	require.Equal(t, after, fresh, "the cells after another pool survive the round trip")
	back.Limit = q.Limit
	moved, err := restored[other.Address].CalcAmountOut(back)
	require.NoError(t, err)
	require.Equal(t, int64(300_000), moved.Gas, "the pool that filled: its other direction's second fill, round-tripped too")
	q.Limit = nil
	fresh, err = restored[p.Address].CalcAmountOut(q)
	require.NoError(t, err)
	require.Equal(t, first, fresh, "round-tripped, with no limit: its cell")
	require.Equal(t, sim.CalculateLimit(), restored[p.Address].CalculateLimit())
}
