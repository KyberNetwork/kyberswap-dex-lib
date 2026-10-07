package ilyris

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// fakeChain answers bitmap, probe and reserve reads from book, the pool's full bin set,
// the way BinPool would.
type fakeChain struct {
	state      RawPoolState
	book       []RawBin // ascending
	guard      RawGuardState
	guardErr   error
	guardBlock uint64
	pools      []FactoryPool
	poolsErr   error
}

func (f *fakeChain) PoolState(context.Context, string) (RawPoolState, error) { return f.state, nil }

func (f *fakeChain) ScanBitmap(_ context.Context, _ string, _ uint64, words []int32,
	probes []binProbe) ([]*big.Int, []binProbeResult, error) {
	bits := make([]*big.Int, len(words))
	for i, w := range words {
		bits[i] = new(big.Int)
		for _, b := range f.book {
			if b.ID>>8 == w {
				bits[i].SetBit(bits[i], int(b.ID&0xff), 1)
			}
		}
	}
	next := make([]binProbeResult, len(probes))
	for i, p := range probes {
		for _, b := range f.book {
			if p.XForY && b.ID <= p.From && b.ReserveY.Sign() > 0 {
				next[i] = binProbeResult{Found: true, ID: b.ID} // keeps the highest
			} else if !p.XForY && b.ID >= p.From && b.ReserveX.Sign() > 0 && !next[i].Found {
				next[i] = binProbeResult{Found: true, ID: b.ID}
			}
		}
	}
	return bits, next, nil
}

func (f *fakeChain) BinReserves(_ context.Context, _ string, _ uint64, ids []int32) ([]RawBin, error) {
	out := make([]RawBin, 0, len(ids))
	for _, b := range f.book {
		for _, id := range ids {
			if b.ID == id {
				out = append(out, b)
			}
		}
	}
	return out, nil
}

func (f *fakeChain) GuardState(_ context.Context, _, _ string, _ int32, blockNumber uint64) (RawGuardState, error) {
	f.guardBlock = blockNumber
	return f.guard, f.guardErr
}

func (f *fakeChain) FactoryPools(_ context.Context, _ string, offset, limit int) ([]FactoryPool, int, error) {
	if f.poolsErr != nil {
		return nil, 0, f.poolsErr
	}
	return f.pools[min(offset, len(f.pools)):min(offset+limit, len(f.pools))], len(f.pools), nil
}

func newFakeChain() *fakeChain {
	return &fakeChain{
		state: RawPoolState{
			BinStepBps: 10, DecimalsX: 18, DecimalsY: 6, ActiveID: 7796, TotalFeeRate: 3_000_000,
			// baseFactor 30000 * binStep 10 * 10 = 3e6, no surcharge
			Fee:         FeeParams{BaseFactor: 30_000, FilterPeriod: 30, DecayPeriod: 600, ReductionFactor: 5_000, MaxVolatilityAccumulator: 350_000, IDReference: 7796},
			MarketGuard: common.HexToAddress("0xDd74981476f81c8e45e962Af6DF886a3c5788816"),
			BlockNumber: 43307616, BlockTimestamp: 1_700_000_000,
		},
		book: []RawBin{
			{ID: 7795, ReserveX: big.NewInt(0), ReserveY: big.NewInt(500_000_000)},
			{ID: 7796, ReserveX: big.NewInt(1e18), ReserveY: big.NewInt(500_000_000)},
		},
	}
}

func refresh(t *testing.T, c *fakeChain, p entity.Pool) (entity.Pool, Extra, error) {
	t.Helper()
	if p.Tokens == nil {
		p.Tokens = []*entity.PoolToken{{Address: tokX}, {Address: tokY}}
	}
	got, err := (&PoolTracker{chain: c}).GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
	var ex Extra
	if err == nil {
		require.NoError(t, json.Unmarshal([]byte(got.Extra), &ex))
	}
	return got, ex, err
}

// A ±N window around active misses liquidity a large swap reaches. The book has Y 200 bins
// down plus one bin past a 5000-bin gap, and X 300 bins up; all must be tracked and fillable.
func TestTrackerLoadsTheWholeBook(t *testing.T) {
	c := newFakeChain()
	c.book = nil
	add := func(id int32, x, y int64) {
		c.book = append(c.book, RawBin{ID: id, ReserveX: big.NewInt(x), ReserveY: big.NewInt(y)})
	}
	add(7796-5000, 0, 1_000_000_000)
	for id := int32(7796 - 200); id < 7796; id++ {
		add(id, 0, 1_000_000)
	}
	add(7796, 1e15, 1_000_000)
	for id := int32(7797); id <= 7796+300; id++ {
		add(id, 1e15, 0)
	}

	p, ex, err := refresh(t, c, entity.Pool{Address: "0xpool", Exchange: DexType, Type: DexType})
	require.NoError(t, err)
	require.Len(t, ex.Bins, len(c.book))
	assert.Equal(t, entity.PoolReserves{"301000000000000000", "1201000000"}, p.Reserves)

	s, err := NewPoolSimulator(p)
	require.NoError(t, err)
	// Sells X through all 201 near bins and on into the isolated one.
	_, err = s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokX, Amount: big.NewInt(3e17)}, TokenOut: tokY})
	require.NoError(t, err)
	// Buys X across ~190 upper bins.
	_, err = s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokY, Amount: big.NewInt(500_000_000)}, TokenOut: tokX})
	require.NoError(t, err)
}

// The simulator derives the fee from feeConfig/feeState; a snapshot where that disagrees with
// getTotalFeeRate() at the same block must be refused and the stored pool kept.
func TestFeeDerivationMismatchIsRefused(t *testing.T) {
	c := newFakeChain()
	c.state.TotalFeeRate++
	before := entity.Pool{Address: "0xpool", Extra: "{}"}
	got, _, err := refresh(t, c, before)
	assert.ErrorIs(t, err, ErrFeeMismatch)
	assert.Equal(t, before.Extra, got.Extra)
}

// The guard is read at the book's block every refresh (its address is owner-mutable), and an
// unreadable guard fails closed: quoting into a reverting swap is worse than skipping a pool.
func TestGuardIsPinnedAndFailsClosed(t *testing.T) {
	c := newFakeChain()
	c.guard = RawGuardState{FreezeEnd: 1_700_000_500}
	p, ex, err := refresh(t, c, entity.Pool{Address: "0xpool"})
	require.NoError(t, err)
	assert.Equal(t, c.state.BlockNumber, c.guardBlock)
	assert.Equal(t, c.state.BlockNumber, p.BlockNumber)
	assert.EqualValues(t, 1_700_000_500, ex.GuardFreezeEnd)
	assert.False(t, ex.GuardSwapsPaused)

	c.guardErr = errors.New("rpc down")
	_, ex, err = refresh(t, c, entity.Pool{Address: "0xpool"})
	require.NoError(t, err)
	assert.True(t, ex.GuardSwapsPaused)
}

// StaticExtra is immutable; a later bad read must not redefine decimals or bin step.
func TestStaticExtraIsWrittenOnlyOnce(t *testing.T) {
	const static = `{"binStepBps":25,"decimalsX":18,"decimalsY":6}`
	got, _, err := refresh(t, newFakeChain(), entity.Pool{Address: "0xpool", StaticExtra: static})
	require.NoError(t, err)
	assert.Equal(t, static, got.StaticExtra)
}

// The cursor only advances past pools that were returned, so a failed page is retried.
func TestListerCursor(t *testing.T) {
	c := &fakeChain{pools: []FactoryPool{{Address: "0xa"}, {Address: "0xb"}, {Address: "0xc"}}}
	u := &PoolsListUpdater{cfg: &Config{DexID: DexType, NewPoolLimit: 2}, chain: c}

	first, md, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, first, 2)
	second, md, err := u.GetNewPools(context.Background(), md)
	require.NoError(t, err)
	require.Len(t, second, 1)
	assert.Equal(t, "0xc", second[0].Address)

	c.poolsErr = errors.New("rpc down")
	_, same, err := u.GetNewPools(context.Background(), md)
	assert.Error(t, err)
	assert.Equal(t, md, same)
}
