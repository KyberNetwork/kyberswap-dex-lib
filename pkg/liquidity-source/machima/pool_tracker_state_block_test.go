package machima

import (
	"math/big"
	"regexp"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/abis"
	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type stateBlockFixture struct {
	t       *testing.T
	stub    *testutil.MulticallStub
	state   *testutil.MethodResponder
	tracker *PoolTracker
}

func newStateBlockFixture(t *testing.T) *stateBlockFixture {
	state := testutil.NewMethodResponder(t, abis.UniswapV3PoolABI, abi.Erc20ABI, clankNowABI, tokenABI)
	stub := testutil.NewMulticallStub(t, state.Respond).At(500, 1)
	tracker, err := NewPoolTracker(&Config{DexID: DexType, ClankNow: "0x5555555555555555555555555555555555555555"},
		stub.Client(), nil)
	require.NoError(t, err)
	f := &stateBlockFixture{t: t, stub: stub, state: state, tracker: tracker}
	f.setChain(1e18, 100)
	return f
}

// setChain is the pool on chain: its liquidity, and the tax of its token. Price and fee never change.
func (f *stateBlockFixture) setChain(liquidity int64, buyTaxBps uint16) {
	f.state.Set("liquidity", big.NewInt(liquidity)).
		Set("slot0", new(big.Int).Lsh(big.NewInt(1), 96), big.NewInt(0), uint16(0), uint16(0), uint16(0), uint32(0), true).
		Set("tickSpacing", big.NewInt(60)).
		Set("fee", big.NewInt(3000)).
		Set("balanceOf", big.NewInt(1e18)).
		Set("getTokenTax", TaxConfig{BuyTaxBps: buyTaxBps, SellTaxBps: 100, HasTax: true}).
		Set("poolDeploymentTime", big.NewInt(1000))
}

func (f *stateBlockFixture) pool() entity.Pool {
	liquidity := big.NewInt(1e18)
	extra := Extra{Extra: uniswapv3.Extra{
		SqrtPriceX96: new(big.Int).Lsh(big.NewInt(1), 96), Tick: big.NewInt(0), Liquidity: liquidity, TickSpacing: 60,
		Ticks: []uniswapv3.Tick{
			{Index: -1980, LiquidityGross: liquidity, LiquidityNet: liquidity},
			{Index: 1980, LiquidityGross: liquidity, LiquidityNet: new(big.Int).Neg(liquidity)},
		},
	}}
	extraBytes, err := json.Marshal(extra)
	require.NoError(f.t, err)
	staticExtra, err := json.Marshal(StaticExtra{Token: tokenAddr, RouterAddress: routerAddr, WETH: wethAddr,
		USDC: usdcAddr, XMA: xmaAddr})
	require.NoError(f.t, err)
	return entity.Pool{
		Address: poolAddr, Exchange: DexType, Type: DexType, BlockNumber: 100,
		Tokens: []*entity.PoolToken{
			{Address: wethAddr, Decimals: 18, Swappable: true}, {Address: tokenAddr, Decimals: 18, Swappable: true}},
		Reserves: entity.PoolReserves{"1000", "1000"}, SwapFee: 3000,
		Extra: string(extraBytes), StaticExtra: string(staticExtra),
	}
}

var stateBlockKey = regexp.MustCompile(`,"sB":\d+`)

// stripStateBlock is the Extra without its sB key, the other bytes untouched.
func stripStateBlock(extra string) string {
	return stateBlockKey.ReplaceAllString(extra, "")
}

// interval is the trigger with no logs: the state is refreshed at latest.
func (f *stateBlockFixture) interval(p entity.Pool) entity.Pool {
	out, err := f.tracker.GetNewPoolState(f.t.Context(), p, poolpkg.GetNewPoolStateParams{})
	require.NoError(f.t, err)
	return out
}

// event is the trigger with logs of the block the chain is at; none of them touches a tick.
func (f *stateBlockFixture) event(p entity.Pool, block uint64) entity.Pool {
	logs := []ethtypes.Log{{Address: common.HexToAddress(poolAddr), Topics: []common.Hash{{1}}, BlockNumber: block}}
	out, err := f.tracker.GetNewPoolState(f.t.Context(), p, poolpkg.GetNewPoolStateParams{Logs: logs})
	require.NoError(f.t, err)
	return out
}

func TestMachimaStateBlock_IntervalPath(t *testing.T) {
	f := newStateBlockFixture(t)

	p := f.interval(f.pool())
	assert.Equal(t, uint64(500), tickspkg.TicksBasedStateBlockOf(p.Extra))
	assert.Equal(t, uint64(100), p.BlockNumber, "the tick cursor is not the state block")

	// nothing changed: the new read block is still stored, and only sB differs
	f.stub.At(510, 1)
	again := f.interval(p)
	assert.Equal(t, uint64(510), tickspkg.TicksBasedStateBlockOf(again.Extra))
	assert.Equal(t, stripStateBlock(p.Extra), stripStateBlock(again.Extra))

	// the tax config changes
	f.stub.At(520, 1)
	f.setChain(1e18, 200)
	p = f.interval(again)
	assert.Equal(t, uint64(520), tickspkg.TicksBasedStateBlockOf(p.Extra))
	assert.Contains(t, p.Extra, `"buyTaxBps":200`)
}

// The event path stores the block of the UniV3 layer's scalar read; applyMachimaState keeps it.
func TestMachimaStateBlock_EventPath(t *testing.T) {
	f := newStateBlockFixture(t)

	p := f.event(f.pool(), 500)
	assert.Equal(t, uint64(500), tickspkg.TicksBasedStateBlockOf(p.Extra))
	assert.Equal(t, uint64(500), p.BlockNumber)

	f.stub.At(510, 1)
	again := f.event(p, 510)
	assert.Equal(t, uint64(510), again.BlockNumber)
	assert.Equal(t, uint64(510), tickspkg.TicksBasedStateBlockOf(again.Extra))
	assert.Equal(t, stripStateBlock(p.Extra), stripStateBlock(again.Extra), "only sB differs")

	f.stub.At(520, 1)
	f.setChain(2e18, 100)
	p = f.event(again, 520)
	assert.Equal(t, uint64(520), tickspkg.TicksBasedStateBlockOf(p.Extra))
}

// FetchPoolTicks keeps the StateBlock and the machima state next to it.
func TestMachimaStateBlock_FetchPoolTicksKeepsIt(t *testing.T) {
	f := newStateBlockFixture(t)
	p := f.interval(f.pool())

	f.stub.At(100, 1)
	f.state.Set("ticks", big.NewInt(4000), big.NewInt(-4000))
	got, err := f.tracker.FetchPoolTicks(t.Context(), p)
	require.NoError(t, err)
	assert.Equal(t, uint64(500), tickspkg.TicksBasedStateBlockOf(got.Extra))
	assert.Contains(t, got.Extra, `"buyTaxBps":100`)
	assert.Contains(t, got.Extra, `"liquidityGross":4000`)
}

func TestMachimaStateBlock_MetaInfo(t *testing.T) {
	f := newStateBlockFixture(t)
	p := f.interval(f.pool())

	sim, err := NewPoolSimulator(p, valueobject.ChainIDBase)
	require.NoError(t, err)
	meta := sim.GetMetaInfo(wethAddr, tokenAddr).(PoolMeta)
	assert.Equal(t, uint64(100), meta.BlockNumber)
	assert.Equal(t, uint64(500), meta.StateBlock)
	out, err := json.Marshal(meta)
	require.NoError(t, err)
	assert.Contains(t, string(out), `"sB":500`)
}

// unmarshalExtra keeps the StateBlock and the machima fields next to the UniV3 Extra.
func TestUnmarshalExtraKeepsStateBlock(t *testing.T) {
	got, err := unmarshalExtra(`{"liquidity":1000,"sqrtPriceX96":79228,"tickSpacing":60,"tick":0,"ticks":[],` +
		`"buyTaxBps":100,"sellTaxBps":200,"hasTax":true,"poolDeploymentTime":1000,"sB":500}`)
	require.NoError(t, err)
	assert.Equal(t, uint64(500), got.StateBlock)
	assert.Equal(t, uint64(60), got.TickSpacing)
	assert.Equal(t, big.NewInt(1000), got.Liquidity)
	assert.Equal(t, uint16(100), got.BuyTaxBps)
	assert.Equal(t, uint16(200), got.SellTaxBps)
	assert.True(t, got.HasTax)
	assert.Equal(t, uint64(1000), got.PoolDeploymentTime)
}

// oldExtra is the Extra the tracker of the release before the StateBlock wrote for the pool of the
// fixture and the chain of the fixture, taken from that release's code (the interval and the event
// path wrote the same bytes). It wrote swap fee 3000 and reserves 1e18, 1e18 with it.
const oldExtra = `{"liquidity":1000000000000000000,"sqrtPriceX96":79228162514264337593543950336,"tickSpacing":60,"tick":0,` +
	`"ticks":[{"index":-1980,"liquidityGross":1000000000000000000,"liquidityNet":1000000000000000000},` +
	`{"index":1980,"liquidityGross":1000000000000000000,"liquidityNet":-1000000000000000000}],` +
	`"buyTaxBps":100,"sellTaxBps":100,"hasTax":true,"poolDeploymentTime":1000}`

// The Extra of an unchanged pool is the one the release before wrote, byte for byte, except the sB
// key, and so are its swap fee and reserves.
func TestMachimaStateBlock_UnchangedStateIdenticalToOldExtraExceptStateBlock(t *testing.T) {
	f := newStateBlockFixture(t)

	got := f.interval(f.pool())
	// sB sits where the embedded UniV3 Extra ends, before the machima fields
	assert.Equal(t, strings.Replace(oldExtra, `,"buyTaxBps"`, `,"sB":500,"buyTaxBps"`, 1), got.Extra,
		"the old Extra and the sB key, no other change")

	old := f.pool()
	old.Extra, old.Reserves = oldExtra, entity.PoolReserves{"1000000000000000000", "1000000000000000000"}
	for name, got := range map[string]entity.Pool{"interval": f.interval(old), "event": f.event(old, 500)} {
		assert.Equal(t, old.Extra, stripStateBlock(got.Extra), name)
		assert.Equal(t, uint64(500), tickspkg.TicksBasedStateBlockOf(got.Extra), name)
		assert.Equal(t, old.SwapFee, got.SwapFee, name)
		assert.Equal(t, old.Reserves, got.Reserves, name)
	}
}
