package integral

import (
	"math/big"
	"regexp"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	sourcePool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var stateBlockKey = regexp.MustCompile(`,"sB":\d+`)

// stripStateBlock is the Extra without its sB key, the other bytes untouched.
func stripStateBlock(extra string) string {
	return stateBlockKey.ReplaceAllString(extra, "")
}

// sbOldExtra is the Extra the tracker of the release before the StateBlock wrote for sbSeedExtra and
// the chain of the fixture (liquidity 1000), taken from that release's code. sbOldFee and sbOldReserves
// are the fee and reserves it wrote with it.
const (
	sbOldExtra = `{"liq":"1000","gS":{"price":"79228162514264337593543950336","tick":0,"lF":100,"pC":0,"cF":0,"un":true},` +
		`"ticks":[{"Index":-60,"LiquidityGross":"1000","LiquidityNet":"1000"},{"Index":60,"LiquidityGross":"1000","LiquidityNet":"-1000"}],"tS":60}`
	sbOldFee = 100
)

// sbOldPool is a pool as the release before the StateBlock stored it.
func sbOldPool() entity.Pool {
	p := stateBlockPool(100, sbOldExtra)
	p.SwapFee = sbOldFee
	p.Reserves = entity.PoolReserves{"1000000000000000000", "1000000000000000000"}
	return p
}

// Every refresh stores the block of its state read. BlockNumber, the cursor, stays.
func TestStateBlock_StoredOnRefresh(t *testing.T) {
	f := newStateBlockFixture(t)

	p := f.update(stateBlockPool(100, sbSeedExtra))
	assert.Equal(t, sbOldExtra[:len(sbOldExtra)-1]+`,"sB":500}`, p.Extra, "the old Extra and the StateBlock, no other change")
	assert.Equal(t, uint64(100), p.BlockNumber)

	f.stub.At(520, 1)
	f.setPool(2000)
	p = f.update(p)
	assert.Equal(t, uint64(520), tickspkg.TicksBasedStateBlockOf(p.Extra))
}

// The Extra of an unchanged pool is the one the release before wrote, byte for byte, except the sB
// key, and so are its swap fee and reserves.
func TestStateBlock_UnchangedStateIdenticalToOldExtraExceptStateBlock(t *testing.T) {
	f := newStateBlockFixture(t)

	got := f.update(sbOldPool())
	want := sbOldPool()
	assert.Equal(t, want.Extra, stripStateBlock(got.Extra))
	assert.Equal(t, uint64(500), tickspkg.TicksBasedStateBlockOf(got.Extra))
	assert.Equal(t, want.SwapFee, got.SwapFee)
	assert.Equal(t, want.Reserves, got.Reserves)

	// a newer read of the same state stores its block: only the sB key differs
	stamped := f.update(stateBlockPool(100, sbSeedExtra))
	f.stub.At(530, 1)
	again := f.update(stamped)
	assert.Equal(t, uint64(530), tickspkg.TicksBasedStateBlockOf(again.Extra))
	assert.Equal(t, stripStateBlock(stamped.Extra), stripStateBlock(again.Extra))
}

// A state read under eth_call overrides exists at no block, so it is not stored.
func TestStateBlock_NotStoredUnderOverrides(t *testing.T) {
	f := newStateBlockFixture(t)
	p := stateBlockPool(100, sbSeedExtra)
	ticksBasedPool := f.ticksBasedPool(p)
	overrides := map[common.Address]gethclient.OverrideAccount{{1}: {}}

	out, err := f.tracker.updateState(t.Context(), p, ticksBasedPool, nil, nil, overrides)
	require.NoError(t, err)
	assert.Contains(t, out.Extra, `"liq":"1000"`)
	assert.Zero(t, tickspkg.TicksBasedStateBlockOf(out.Extra))

	// a pool that has a StateBlock loses it under overrides
	p = f.update(p)
	require.Equal(t, uint64(500), tickspkg.TicksBasedStateBlockOf(p.Extra))
	out, err = f.tracker.updateState(t.Context(), p, f.ticksBasedPool(p), nil, nil, overrides)
	require.NoError(t, err)
	assert.Zero(t, tickspkg.TicksBasedStateBlockOf(out.Extra))
}

func TestStateBlock_BootstrapStoresReadBlock(t *testing.T) {
	f := newStateBlockFixture(t)
	p, err := f.tracker.BootstrapPoolState(t.Context(), stateBlockPool(100, ""), sourcePool.GetNewPoolStateParams{})
	require.NoError(t, err)
	assert.Equal(t, uint64(500), tickspkg.TicksBasedStateBlockOf(p.Extra))
	assert.Equal(t, uint64(500), p.BlockNumber,
		"dated by the head its state is read at, which its ticks are read again at")
}

// A refetch of the stored ticks keeps the StateBlock: it reads the ticks, not the scalar state.
func TestStateBlock_FetchPoolTicksKeepsIt(t *testing.T) {
	f := newStateBlockFixture(t)
	p := f.update(stateBlockPool(100, sbSeedExtra))
	require.Equal(t, uint64(500), tickspkg.TicksBasedStateBlockOf(p.Extra))

	f.stub.At(100, 1) // the ticks are read at the pool's BlockNumber
	f.state.Set("ticks", big.NewInt(4000), big.NewInt(4000), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0))
	got, err := f.tracker.FetchPoolTicks(t.Context(), p)
	require.NoError(t, err)
	assert.Equal(t, uint64(500), tickspkg.TicksBasedStateBlockOf(got.Extra))
	assert.Contains(t, got.Extra, `"LiquidityGross":"4000"`, "the ticks were read again")
}

// The meta of a pool built from the stored Extra has the key sB: max(StateBlock, BlockNumber).
func TestStateBlock_MetaInfo(t *testing.T) {
	f := newStateBlockFixture(t)
	p := f.update(stateBlockPool(100, sbSeedExtra))
	p.Reserves = entity.PoolReserves{"1000000000000000000", "1000000000000000000"}
	p.StaticExtra = `{}`

	meta := func(p entity.Pool) (PoolMeta, string) {
		sim, err := NewPoolSimulator(p)
		require.NoError(t, err)
		info := sim.GetMetaInfo(p.Tokens[0].Address, p.Tokens[1].Address)
		out, err := json.Marshal(info)
		require.NoError(t, err)
		return info.(PoolMeta), string(out)
	}

	m, out := meta(p)
	assert.Equal(t, uint64(100), m.BlockNumber, "blockNumber is still the cursor")
	assert.Equal(t, uint64(500), m.StateBlock)
	assert.Contains(t, out, `"sB":500`)

	p.BlockNumber = 700
	m, _ = meta(p)
	assert.Equal(t, uint64(700), m.StateBlock, "the cursor passed the read")

	old := sbOldPool()
	old.StaticExtra = `{}`
	m, out = meta(old)
	assert.Zero(t, m.StateBlock)
	assert.NotContains(t, out, "sB", "omitted when unknown")
}
