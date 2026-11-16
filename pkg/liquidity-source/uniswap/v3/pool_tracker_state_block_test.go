package uniswapv3

import (
	"math/big"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/KyberNetwork/int256"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/abis"
	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const seedExtra = `{"liquidity":"1000","sqrtPriceX96":"79228162514264337593543950336","tickSpacing":60,"tick":0,` +
	`"ticks":[{"index":-60,"liquidityGross":"1000","liquidityNet":"1000"},` +
	`{"index":60,"liquidityGross":"1000","liquidityNet":"-1000"}]}`

type stateBlockFixture struct {
	t       *testing.T
	stub    *testutil.MulticallStub
	state   *testutil.MethodResponder
	tracker *Tracker
}

// newStateBlockFixture is a tracker whose chain answers its multicalls at the block of At.
func newStateBlockFixture(t *testing.T) *stateBlockFixture {
	state := testutil.NewMethodResponder(t, abis.UniswapV3PoolABI, abi.Erc20ABI)
	stub := testutil.NewMulticallStub(t, state.Respond).At(500, 1)
	gql := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"ticks":[]}}`))
	}))
	t.Cleanup(gql.Close)
	tracker, err := NewTracker(&Config{DexID: DexTypeUniswapV3}, stub.Client(), graphqlpkg.NewClient(gql.URL))
	require.NoError(t, err)
	f := &stateBlockFixture{t: t, stub: stub, state: state, tracker: tracker}
	f.setPool(1000)
	return f
}

// setPool is the state of the pool on chain: its liquidity, with a price and fee that never change.
func (f *stateBlockFixture) setPool(liquidity int64) {
	f.state.Set("liquidity", big.NewInt(liquidity)).
		Set("slot0", new(big.Int).Lsh(big.NewInt(1), 96), big.NewInt(0), uint16(0), uint16(0), uint16(0), uint32(0), true).
		Set("tickSpacing", big.NewInt(60)).
		Set("fee", big.NewInt(3000)).
		Set("balanceOf", big.NewInt(1e18))
}

func stateBlockPool(blockNumber uint64, extra string) entity.Pool {
	return entity.Pool{
		Address:  "0x1111111111111111111111111111111111111111",
		Exchange: DexTypeUniswapV3, Type: DexTypeUniswapV3,
		BlockNumber: blockNumber,
		Tokens: []*entity.PoolToken{
			{Address: "0x2222222222222222222222222222222222222222", Decimals: 18, Swappable: true},
			{Address: "0x3333333333333333333333333333333333333333", Decimals: 18, Swappable: true},
		},
		Extra: extra,
	}
}

// update is one event-driven write of p: with no logs the ticks stay, the scalar state is read.
func (f *stateBlockFixture) update(p entity.Pool) entity.Pool {
	ticksBasedPool, err := tickspkg.NewTicksBasedPool(p)
	require.NoError(f.t, err)
	out, err := f.tracker.updateState(f.t.Context(), p, ticksBasedPool, nil, nil)
	require.NoError(f.t, err)
	return out
}

// Every refresh stores the block of its state read. BlockNumber, the tick cursor, stays.
func TestStateBlock_StoredOnRefresh(t *testing.T) {
	f := newStateBlockFixture(t)

	p := f.update(stateBlockPool(100, seedExtra))
	assert.Equal(t, uint64(500), tickspkg.TicksBasedStateBlockOf(p.Extra))
	assert.Equal(t, uint64(100), p.BlockNumber, "the tick cursor is not the state block")

	f.stub.At(520, 1)
	f.setPool(2000)
	p = f.update(p)
	assert.Equal(t, uint64(520), tickspkg.TicksBasedStateBlockOf(p.Extra))
	assert.Equal(t, uint64(100), p.BlockNumber)
}

// A refresh of a pool whose state did not change still stores the block of the read: sB tells when the
// state was last read.
func TestStateBlock_UnchangedStateStoresNewReadBlock(t *testing.T) {
	f := newStateBlockFixture(t)
	p := f.update(stateBlockPool(100, seedExtra))

	f.stub.At(530, 1)
	again := f.update(p)
	assert.Equal(t, uint64(530), tickspkg.TicksBasedStateBlockOf(again.Extra))
	assert.Equal(t, stripStateBlock(p.Extra), stripStateBlock(again.Extra), "no other change")
	assert.Equal(t, p.SwapFee, again.SwapFee)
	assert.Equal(t, p.Reserves, again.Reserves)
}

var stateBlockKey = regexp.MustCompile(`,"sB":\d+`)

// stripStateBlock is the Extra without its sB key, the other bytes untouched.
func stripStateBlock(extra string) string {
	return stateBlockKey.ReplaceAllString(extra, "")
}

func TestStateBlock_BootstrapStoresReadBlock(t *testing.T) {
	f := newStateBlockFixture(t)
	p, err := f.tracker.BootstrapPoolState(t.Context(), stateBlockPool(100, ""), poolpkg.GetNewPoolStateParams{})
	require.NoError(t, err)
	assert.Equal(t, uint64(500), tickspkg.TicksBasedStateBlockOf(p.Extra))
	assert.Equal(t, uint64(100), p.BlockNumber)
}

// A refetch of the stored ticks keeps the StateBlock: it reads the ticks, not the scalar state.
func TestStateBlock_FetchPoolTicksKeepsIt(t *testing.T) {
	f := newStateBlockFixture(t)
	p := f.update(stateBlockPool(100, seedExtra))
	require.Equal(t, uint64(500), tickspkg.TicksBasedStateBlockOf(p.Extra))

	f.stub.At(100, 1) // the ticks are read at the pool's BlockNumber
	f.state.Set("ticks", big.NewInt(4000), big.NewInt(-4000))
	got, err := f.tracker.FetchPoolTicks(t.Context(), p)
	require.NoError(t, err)
	assert.Equal(t, uint64(500), tickspkg.TicksBasedStateBlockOf(got.Extra))
	assert.Contains(t, got.Extra, `"liquidityGross":4000`, "the ticks were read again")
}

// The GetMetaInfo of a pool built from the stored Extra exposes the key sB: max(StateBlock, BlockNumber).
func TestStateBlock_MetaInfo(t *testing.T) {
	f := newStateBlockFixture(t)
	p := f.update(stateBlockPool(100, seedExtra))

	meta := func(p entity.Pool) (PoolMeta, string) {
		sim, err := NewPoolSimulator(p, valueobject.ChainIDEthereum)
		require.NoError(t, err)
		info := sim.GetMetaInfo(p.Tokens[0].Address, p.Tokens[1].Address)
		out, err := json.Marshal(info)
		require.NoError(t, err)
		return info.(PoolMeta), string(out)
	}

	m, out := meta(p)
	assert.Equal(t, uint64(100), m.BlockNumber, "blockNumber is still the tick cursor")
	assert.Equal(t, uint64(500), m.StateBlock)
	assert.Contains(t, out, `"sB":500`)

	p.BlockNumber = 700 // the cursor passed the read: the state is valid at the cursor at least
	m, _ = meta(p)
	assert.Equal(t, uint64(700), m.StateBlock)

	p.Extra = stripStateBlock(p.Extra)
	m, out = meta(p)
	assert.Zero(t, m.StateBlock)
	assert.NotContains(t, out, "sB", "omitted when unknown")
}

// Extra converted from the tolerant uint256 shape keeps the StateBlock, and the fields next to it.
func TestExtraTickU256ToExtraKeepsStateBlock(t *testing.T) {
	src := ExtraTickU256{
		Liquidity:            uint256.NewInt(1000),
		SqrtPriceX96:         uint256.NewInt(79228),
		TickSpacing:          60,
		Tick:                 new(int),
		Ticks:                []TickU256{{Index: 60, LiquidityGross: uint256.NewInt(5), LiquidityNet: int256.NewInt(-5)}},
		TicksBasedStateBlock: tickspkg.TicksBasedStateBlock{StateBlock: 500},
	}

	out := src.ToExtra()
	assert.Equal(t, uint64(500), out.StateBlock)
	assert.Equal(t, uint64(60), out.TickSpacing)
	assert.Equal(t, big.NewInt(1000), out.Liquidity)
	assert.Equal(t, big.NewInt(79228), out.SqrtPriceX96)
	require.Len(t, out.Ticks, 1)
	assert.Equal(t, 60, out.Ticks[0].Index)
	assert.Equal(t, big.NewInt(-5), out.Ticks[0].LiquidityNet)
}

// A failed call inside the multicall (here currentFee, which most pools lack) must not fail the
// fetch: the multicall answers per call, with its block.
func TestStateBlock_FailedCallInsideMulticallDoesNotFailFetch(t *testing.T) {
	f := newStateBlockFixture(t)
	before := len(f.stub.Targets())
	p := f.update(stateBlockPool(100, seedExtra))
	assert.Equal(t, uint64(500), tickspkg.TicksBasedStateBlockOf(p.Extra))
	assert.Contains(t, p.Extra, `"liquidity":1000`)
	assert.Len(t, f.stub.Targets()[before:], 1, "one multicall")
}

// The tryBlockAndAggregate result decodes a slot0 of any shape like tryAggregate did: a slipstream
// pool (6 words) must not be read as the 7-word standard shape.
func TestStateBlock_SlipstreamSlot0StillDecodes(t *testing.T) {
	f := newStateBlockFixture(t)
	packed, err := abis.Slot0SlipstreamABI.Methods["slot0"].Outputs.Pack(
		new(big.Int).Lsh(big.NewInt(3), 96), big.NewInt(7), uint16(0), uint16(0), uint16(0), true)
	require.NoError(t, err)
	f.state.SetRaw("slot0", packed)

	p := f.update(stateBlockPool(100, seedExtra))
	assert.Equal(t, uint64(500), tickspkg.TicksBasedStateBlockOf(p.Extra))
	assert.Contains(t, p.Extra, `"tick":7`)
	assert.Contains(t, p.Extra, `"sqrtPriceX96":237684487542793012780631851008`)
}

// oldExtra is the Extra the tracker of the release before the StateBlock wrote for seedExtra and the
// chain of the fixture (liquidity 1000), taken from that release's code, with the fee and reserves
// it wrote with it.
const oldExtra = `{"liquidity":1000,"sqrtPriceX96":79228162514264337593543950336,"tickSpacing":60,"tick":0,` +
	`"ticks":[{"index":-60,"liquidityGross":1000,"liquidityNet":1000},{"index":60,"liquidityGross":1000,"liquidityNet":-1000}]}`

// The Extra of an unchanged pool is the one the release before wrote, byte for byte, except the
// sB key, and so are its swap fee and reserves.
func TestStateBlock_UnchangedStateIdenticalToOldExtraExceptStateBlock(t *testing.T) {
	f := newStateBlockFixture(t)

	got := f.update(stateBlockPool(100, seedExtra))
	assert.Equal(t, oldExtra[:len(oldExtra)-1]+`,"sB":500}`, got.Extra, "the old Extra and the sB key, no other change")

	old := stateBlockPool(100, oldExtra)
	old.SwapFee = 3000
	old.Reserves = entity.PoolReserves{"1000000000000000000", "1000000000000000000"}
	got = f.update(old)
	assert.Equal(t, oldExtra, stripStateBlock(got.Extra), "the old Extra except the sB key")
	assert.Equal(t, uint64(500), tickspkg.TicksBasedStateBlockOf(got.Extra))
	assert.Equal(t, old.SwapFee, got.SwapFee)
	assert.Equal(t, old.Reserves, got.Reserves)
}

// A refetch of the ticks goes through ExtraTickU256.ToExtra, which replaced a hand-written copy: an
// Extra of the release before comes out byte for byte.
func TestExtraTickU256ToExtraKeepsOldBytes(t *testing.T) {
	var u ExtraTickU256
	require.NoError(t, json.Unmarshal([]byte(oldExtra), &u))
	out, err := json.Marshal(u.ToExtra())
	require.NoError(t, err)
	assert.Equal(t, oldExtra, string(out))
}
