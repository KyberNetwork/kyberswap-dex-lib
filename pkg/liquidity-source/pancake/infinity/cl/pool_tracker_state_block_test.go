package cl

import (
	"context"
	"math/big"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync/atomic"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	factory "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/cl/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/shared"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	sbManager  = "0xa0ffb9c1ce1fe56963b0321b32e7a0302114058b"
	sbPoolID   = "0x0000000000000000000000000000000000000000000000000000000000000def"
	sbHookAddr = "0x00000000000000000000000000000000000d2001"
)

// sbHookBlock is the block the hook of a state block test was asked to read at, 0 for latest.
var sbHookBlock atomic.Uint64

// sbHook reads nothing; it records the block the tracker pinned its hook reads at.
type sbHook struct{ Hook }

func (h sbHook) Track(_ context.Context, param *HookParam) ([]byte, error) {
	sbHookBlock.Store(0)
	if param.BlockNumber != nil {
		sbHookBlock.Store(param.BlockNumber.Uint64())
	}
	return h.Hook.Track(context.Background(), param)
}

var _ = RegisterHooksFactory(func(p *HookParam) Hook { return sbHook{NewBaseHook(DexType, p)} },
	common.HexToAddress(sbHookAddr))

type stateBlockFixture struct {
	t       *testing.T
	stub    *testutil.MulticallStub
	state   *testutil.MethodResponder
	tracker *PoolTracker
}

func newStateBlockFixture(t *testing.T) *stateBlockFixture {
	state := testutil.NewMethodResponder(t, shared.CLPoolManagerABI)
	stub := testutil.NewMulticallStub(t, state.Respond).At(500, 1)
	gql := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"ticks":[]}}`)) // the subgraph has no ticks: they do not matter here
	}))
	t.Cleanup(gql.Close)
	f := &stateBlockFixture{t: t, stub: stub, state: state,
		tracker: NewPoolTracker(&Config{CLPoolManagerAddress: sbManager, DexID: DexType, ChainID: 1}, stub.Client(),
			graphqlpkg.NewClient(gql.URL))}
	f.setChain(1000)
	sbHookBlock.Store(0)
	return f
}

func (f *stateBlockFixture) setChain(liquidity int64) {
	f.state.Set("getLiquidity", big.NewInt(liquidity)).
		Set("getSlot0", new(big.Int).Lsh(big.NewInt(1), 96), big.NewInt(0), uint32(0), uint32(3000))
}

func (f *stateBlockFixture) pool() entity.Pool {
	extra, err := json.Marshal(Extra{Extra: &uniswapv3.Extra{
		Liquidity: big.NewInt(1000), SqrtPriceX96: new(big.Int).Lsh(big.NewInt(1), 96), TickSpacing: 60, Tick: big.NewInt(0),
		Ticks: []Tick{
			{Index: -60, LiquidityGross: big.NewInt(1000), LiquidityNet: big.NewInt(1000)},
			{Index: 60, LiquidityGross: big.NewInt(1000), LiquidityNet: big.NewInt(-1000)},
		},
	}})
	require.NoError(f.t, err)
	static, err := json.Marshal(StaticExtra{TickSpacing: 60, Fee: 3000, Parameters: "0x0", PoolManagerAddress: common.HexToAddress(sbManager),
		HooksAddress: common.HexToAddress(sbHookAddr)})
	require.NoError(f.t, err)
	return entity.Pool{
		Address: sbPoolID, Exchange: DexType, Type: DexType, BlockNumber: 100, SwapFee: 3000,
		Tokens: []*entity.PoolToken{
			{Address: "0x2222222222222222222222222222222222222222", Decimals: 18, Swappable: true},
			{Address: "0x3333333333333333333333333333333333333333", Decimals: 18, Swappable: true}},
		Extra: string(extra), StaticExtra: string(static),
	}
}

var stateBlockKey = regexp.MustCompile(`,"sB":\d+`)

// stripStateBlock is the Extra without its sB key, the other bytes untouched.
func stripStateBlock(extra string) string {
	return stateBlockKey.ReplaceAllString(extra, "")
}

func (f *stateBlockFixture) update(p entity.Pool) entity.Pool {
	ticksBasedPool, err := tickspkg.NewTicksBasedPool(p)
	require.NoError(f.t, err)
	out, err := f.tracker.updateState(f.t.Context(), p, ticksBasedPool, nil, nil)
	require.NoError(f.t, err)
	return out
}

func TestStateBlock_StoredOnRefresh(t *testing.T) {
	f := newStateBlockFixture(t)

	p := f.update(f.pool())
	assert.Equal(t, uint64(500), tickspkg.TicksBasedStateBlockOf(p.Extra))
	assert.Equal(t, uint64(100), p.BlockNumber, "the tick cursor is not the state block")
	assert.Equal(t, uint64(500), sbHookBlock.Load(), "the hook reads are pinned at the block of the state read")

	f.stub.At(520, 1)
	f.setChain(2000)
	p = f.update(p)
	assert.Equal(t, uint64(520), tickspkg.TicksBasedStateBlockOf(p.Extra))
	assert.Equal(t, uint64(520), sbHookBlock.Load())
}

// A refresh of a pool whose state did not change still stores the block of the read: sB tells when the
// state was last read.
func TestStateBlock_UnchangedStateStoresNewReadBlock(t *testing.T) {
	f := newStateBlockFixture(t)
	p := f.update(f.pool())

	f.stub.At(530, 1)
	again := f.update(p)
	assert.Equal(t, uint64(530), tickspkg.TicksBasedStateBlockOf(again.Extra))
	assert.Equal(t, uint64(530), sbHookBlock.Load())
	assert.Equal(t, stripStateBlock(p.Extra), stripStateBlock(again.Extra), "no other change")
}

func TestStateBlock_BootstrapStoresReadBlock(t *testing.T) {
	f := newStateBlockFixture(t)
	p := f.pool()
	p.Extra = ""
	p, err := f.tracker.BootstrapPoolState(t.Context(), p, poolpkg.GetNewPoolStateParams{})
	require.NoError(t, err)
	assert.Equal(t, uint64(500), tickspkg.TicksBasedStateBlockOf(p.Extra))
	assert.Equal(t, uint64(100), p.BlockNumber)
	assert.Equal(t, uint64(500), sbHookBlock.Load())
}

func TestStateBlock_MetaInfo(t *testing.T) {
	f := newStateBlockFixture(t)
	p := f.update(f.pool())

	meta := func(p entity.Pool) (PoolMetaInfo, string) {
		sim, err := NewPoolSimulator(p, valueobject.ChainIDEthereum)
		require.NoError(t, err)
		info := sim.GetMetaInfo(p.Tokens[0].Address, p.Tokens[1].Address)
		out, err := json.Marshal(info)
		require.NoError(t, err)
		return info.(PoolMetaInfo), string(out)
	}
	m, out := meta(p)
	assert.Equal(t, uint64(100), m.BlockNumber)
	assert.Equal(t, uint64(500), m.StateBlock)
	assert.Contains(t, out, `"sB":500`)

	p.Extra = stripStateBlock(p.Extra)
	m, out = meta(p)
	assert.Zero(t, m.StateBlock)
	assert.NotContains(t, out, "sB")
}

// oldExtra is the Extra the tracker of the release before the StateBlock wrote for the pool of the
// fixture and the chain of the fixture, taken from that release's code. It wrote swap fee 3000 and
// reserves 2, 2 (the estimate from the ticks) with it.
const oldExtra = `{"liquidity":1000,"sqrtPriceX96":79228162514264337593543950336,"tickSpacing":60,"tick":0,` +
	`"ticks":[{"index":-60,"liquidityGross":1000,"liquidityNet":1000},{"index":60,"liquidityGross":1000,"liquidityNet":-1000}]}`

// The Extra of an unchanged pool is the one the release before wrote, byte for byte, except the sB
// key, and so are its swap fee and reserves.
func TestStateBlock_UnchangedStateIdenticalToOldExtraExceptStateBlock(t *testing.T) {
	f := newStateBlockFixture(t)

	got := f.update(f.pool())
	assert.Equal(t, oldExtra[:len(oldExtra)-1]+`,"sB":500}`, got.Extra, "the old Extra and the StateBlock, no other change")

	old := f.pool()
	old.Extra, old.Reserves = oldExtra, entity.PoolReserves{"2", "2"}
	got = f.update(old)
	assert.Equal(t, old.Extra, stripStateBlock(got.Extra))
	assert.Equal(t, uint64(500), tickspkg.TicksBasedStateBlockOf(got.Extra))
	assert.Equal(t, old.SwapFee, got.SwapFee)
	assert.Equal(t, old.Reserves, got.Reserves)
}

// A refetch of the stored ticks keeps the StateBlock: it reads the ticks, not the scalar state.
func TestStateBlock_FetchPoolTicksKeepsIt(t *testing.T) {
	f := newStateBlockFixture(t)
	p := f.update(f.pool())
	require.Equal(t, uint64(500), tickspkg.TicksBasedStateBlockOf(p.Extra))

	f.stub.At(100, 1) // the ticks are read at the pool's BlockNumber
	f.state.Set("getPoolTickInfo", factory.TickInfo{
		LiquidityGross: big.NewInt(4000), LiquidityNet: big.NewInt(4000),
		FeeGrowthOutside0X128: big.NewInt(0), FeeGrowthOutside1X128: big.NewInt(0)})
	got, err := f.tracker.FetchPoolTicks(t.Context(), p)
	require.NoError(t, err)
	assert.Equal(t, uint64(500), tickspkg.TicksBasedStateBlockOf(got.Extra))
	assert.Contains(t, got.Extra, `"liquidityGross":4000`, "the ticks were read again")
}
