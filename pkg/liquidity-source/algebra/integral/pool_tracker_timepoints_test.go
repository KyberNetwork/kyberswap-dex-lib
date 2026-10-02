package integral

import (
	"cmp"
	"context"
	"math"
	"math/big"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/algebra"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const multicallAggregateABI = `[{"name":"aggregate","type":"function","stateMutability":"view",
"inputs":[{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],
"outputs":[{"name":"blockNumber","type":"uint256"},{"name":"returnData","type":"bytes[]"}]}]`

const (
	tpNow     = uint32(1737563754)
	tpSpacing = uint32(600)
	tpOldTs   = uint32(1600000000) // ts of data left over from a previous ring buffer cycle
)

// fakeChain is a minimal ring buffer of plugin timepoints served through a fake eth_call/multicall endpoint.
type fakeChain struct {
	mu        sync.Mutex
	slots     map[uint16]uint32 // index -> block timestamp (0 = uninitialized). Other fields derive from ts.
	index     uint16
	override  map[uint16]Timepoint // exact on-chain values for some slots, instead of derived ones
	requested []uint16
}

// newFakeChain creates a chain whose slots all hold previous-cycle data, except for the last n slots ending at index
// which were written every tpSpacing seconds, the last one at tpNow.
func newFakeChain(index uint16, n int, full bool) *fakeChain {
	c := &fakeChain{slots: map[uint16]uint32{}, index: index}
	if full {
		for i := range math.MaxUint16 + 1 {
			c.slots[uint16(i)] = tpOldTs + uint32(uint16(i)-index-1) // oldest right after the head
		}
	}
	for i := range n {
		c.slots[index-uint16(i)] = tpNow - uint32(i)*tpSpacing
	}
	return c
}

// write appends k new timepoints, each tpSpacing after the previous one, the last one at ts of the new head.
func (c *fakeChain) write(k int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	head := c.slots[c.index]
	for range k {
		c.index++
		head += tpSpacing
		c.slots[c.index] = head
	}
}

func (c *fakeChain) headTs() uint32 { return c.slots[c.index] }

func (c *fakeChain) takeRequested() []uint16 {
	c.mu.Lock()
	defer c.mu.Unlock()
	r := slices.Clone(c.requested)
	c.requested = nil
	slices.Sort(r)
	return r
}

type ethAPI struct{ chain *fakeChain }

func (e *ethAPI) Call(args struct {
	To    *common.Address `json:"to"`
	Data  *hexutil.Bytes  `json:"data"`
	Input *hexutil.Bytes  `json:"input"`
}, _ rpc.BlockNumberOrHash, _ *map[common.Address]any) (hexutil.Bytes, error) {
	multicallABI := lo.Must(abi.JSON(strings.NewReader(multicallAggregateABI)))
	data := cmp.Or(args.Input, args.Data)
	var in struct {
		Calls []struct {
			Target   common.Address
			CallData []byte
		}
	}
	vals, err := multicallABI.Methods["aggregate"].Inputs.Unpack((*data)[4:])
	if err != nil {
		return nil, err
	}
	if err = multicallABI.Methods["aggregate"].Inputs.Copy(&in, vals); err != nil {
		return nil, err
	}

	e.chain.mu.Lock()
	defer e.chain.mu.Unlock()
	rets := make([][]byte, 0, len(in.Calls))
	for _, call := range in.Calls {
		m, err := basePluginV2ABI.MethodById(call.CallData[:4])
		if err != nil {
			return nil, err
		}
		params, err := m.Inputs.Unpack(call.CallData[4:])
		if err != nil {
			return nil, err
		}
		idx := uint16(params[0].(*big.Int).Uint64())
		e.chain.requested = append(e.chain.requested, idx)
		ts := e.chain.slots[idx]
		// a distinct tick per timestamp makes any mix-up between cycles visible in the result
		tp := TimepointRPC{Initialized: ts != 0, BlockTimestamp: ts, TickCumulative: big.NewInt(int64(ts)),
			VolatilityCumulative: big.NewInt(int64(ts) + 1), Tick: big.NewInt(int64(ts % 100)),
			AverageTick: big.NewInt(int64(ts % 100))}
		if o, ok := e.chain.override[idx]; ok {
			tp = TimepointRPC{Initialized: o.Initialized, BlockTimestamp: o.BlockTimestamp,
				TickCumulative: big.NewInt(o.TickCumulative), VolatilityCumulative: o.VolatilityCumulative.ToBig(),
				Tick: big.NewInt(int64(o.Tick)), AverageTick: big.NewInt(int64(o.AverageTick)),
				WindowStartIndex: o.WindowStartIndex}
		}
		out, err := m.Outputs.Pack(tp.Initialized, tp.BlockTimestamp, tp.TickCumulative, tp.VolatilityCumulative,
			tp.Tick, tp.AverageTick, tp.WindowStartIndex)
		if err != nil {
			return nil, err
		}
		rets = append(rets, out)
	}
	out, err := multicallABI.Methods["aggregate"].Outputs.Pack(big.NewInt(1), rets)
	return out, err
}

func newTrackerOnChain(t *testing.T, chain *fakeChain) *PoolTracker {
	srv := rpc.NewServer()
	require.NoError(t, srv.RegisterName("eth", &ethAPI{chain}))
	client := rpc.DialInProc(srv)
	t.Cleanup(func() { client.Close(); srv.Stop() })
	return &PoolTracker{PoolTracker: algebra.PoolTracker[Timepoint, TimepointRPC]{
		EthrpcClient: ethrpc.NewWithClient(ethclient.NewClient(client)),
	}}
}

func (c *fakeChain) fetch(t *testing.T, tr *PoolTracker, cache map[uint16]Timepoint) map[uint16]Timepoint {
	res, err := tr.getTimepoints(context.Background(), "0x3E6e46Ea6E7f5D659dD1b3aB889cb8Bc199A072F", nil, c.index,
		c.headTs(), cache, nil)
	require.NoError(t, err)
	return res
}

func requireSameTimepoints(t *testing.T, want, got map[uint16]Timepoint) {
	t.Helper()
	require.Equal(t, len(want), len(got))
	for idx, w := range want {
		g, ok := got[idx]
		require.True(t, ok, "missing timepoint %d", idx)
		require.Equal(t, w.BlockTimestamp, g.BlockTimestamp, "timepoint %d", idx)
		require.Equal(t, w.TickCumulative, g.TickCumulative, "timepoint %d", idx)
	}
}

func TestGetTimepoints_StaleCache(t *testing.T) {
	t.Parallel()
	const yesterday = tpNow - WINDOW

	tests := []struct {
		name      string
		prevIndex uint16
		newSlots  int
		full      bool // ring buffer already wrapped once, so slots ahead of the head hold old-cycle data
	}{
		{"single new slot, buffer wrapped", 1000, 1, true},
		{"several new slots across uint16 wrap", 65530, 7, true},
		{"several new slots, no wrap", 1000, 5, true},
		{"new slots, buffer not full yet", 1000, 3, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			chain := newFakeChain(tc.prevIndex, 200, tc.full)
			tr := newTrackerOnChain(t, chain)

			cache := chain.fetch(t, tr, nil)
			chain.takeRequested()
			if tc.full { // the extra fetches cached the old-cycle slots just ahead of the head
				require.Equal(t, tpOldTs, cache[tc.prevIndex+1].BlockTimestamp)
			}

			chain.write(tc.newSlots)
			require.NotEqual(t, tc.prevIndex, chain.index)
			got := chain.fetch(t, tr, cache)
			requested := chain.takeRequested()

			// every slot written since the last refresh must be re-fetched
			for i := 1; i <= tc.newSlots; i++ {
				idx := tc.prevIndex + uint16(i)
				require.Contains(t, requested, idx)
				require.Equal(t, chain.slots[idx], got[idx].BlockTimestamp, "slot %d", idx)
			}
			require.Equal(t, chain.headTs(), got[chain.index].BlockTimestamp)
			// unchanged slots are not fetched again
			for i := range 100 {
				require.NotContains(t, requested, tc.prevIndex-uint16(i), "unchanged slot refetched")
			}
			require.LessOrEqual(t, len(requested), tc.newSlots+8, "refetch must stay bounded: %v", requested)

			// the result matches a fresh fetch from scratch, and nothing older than the window is the current slot
			fresh := newFakeChain(0, 0, false)
			fresh.slots, fresh.index = chain.slots, chain.index
			want := fresh.fetch(t, newTrackerOnChain(t, fresh), nil)
			requireSameTimepoints(t, want, got)
			require.GreaterOrEqual(t, got[chain.index].BlockTimestamp, uint32(yesterday))
		})
	}
}

func TestGetTimepoints_UnchangedNotRefetched(t *testing.T) {
	t.Parallel()
	chain := newFakeChain(65530, 200, true)
	tr := newTrackerOnChain(t, chain)

	cache := chain.fetch(t, tr, nil)
	first := chain.takeRequested()
	require.NotEmpty(t, first)

	got := chain.fetch(t, tr, cache)
	require.Empty(t, chain.takeRequested(), "nothing changed on chain, so nothing should be fetched")
	requireSameTimepoints(t, cache, got)
}

// TestGetTimepoints_StaleCacheOverquote reproduces thena-fusion-v3 pool 0x7491c04d...fb7dcc1 on BSC (USDC -> USDT).
// The cached timepoint at the current index 41557 is from a previous cycle of the ring buffer (June), while on-chain
// it was rewritten at 1790916437. Plugin getCurrentFee() = 725, so the quote for 10,000 USDC must match the one
// computed with the on-chain fee, instead of the ~7 bps better one derived from stale timepoints.
func TestGetTimepoints_StaleCacheOverquote(t *testing.T) {
	const (
		now         = uint32(1790916437)
		headIndex   = uint16(41557)
		staleQuote  = "10003070347660032501178" // quote from the stale stored pool
		onchainFee  = "9996068081662022905388"  // same pool with the on-chain fee 725 (no adaptive fee override)
		windowStart = uint16(41256)
	)
	oldBlockTimestamp := blockTimestamp
	blockTimestamp = func() uint32 { return now }
	t.Cleanup(func() { blockTimestamp = oldBlockTimestamp })

	var ep entity.Pool
	require.NoError(t, json.Unmarshal([]byte(stalePoolFixture), &ep))
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(ep.Extra), &extra))
	require.Equal(t, uint32(1780562536), extra.Timepoints[headIndex].BlockTimestamp, "fixture must hold the stale slot")
	require.Equal(t, now, extra.VolatilityOracle.LastTimepointTimestamp)

	quote := func(extra Extra) string {
		ep := ep
		ep.Extra = string(lo.Must(json.Marshal(extra)))
		sim := lo.Must(NewPoolSimulator(ep))
		res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: ep.Tokens[1].Address,
				Amount: new(big.Int).Mul(big.NewInt(10000), bignumber.TenPowInt(18))},
			TokenOut: ep.Tokens[0].Address,
		})
		require.NoError(t, err)
		return res.TokenAmountOut.Amount.String()
	}
	require.Equal(t, staleQuote, quote(extra), "stale cache overquotes")

	// on-chain: exact values for the slots the simulator reads, synthetic evenly spaced ones in between
	var onchain map[uint16]Timepoint
	require.NoError(t, json.Unmarshal([]byte(onchainTimepointsFixture), &onchain))
	chain := &fakeChain{slots: map[uint16]uint32{}, index: headIndex, override: onchain}
	first, last := onchain[41257], onchain[41556]
	for idx := uint16(41257); idx <= 41556; idx++ {
		chain.slots[idx] = first.BlockTimestamp + uint32(idx-41257)*(last.BlockTimestamp-first.BlockTimestamp)/(41556-41257)
	}
	for idx, tp := range onchain {
		chain.slots[idx] = tp.BlockTimestamp
	}
	tr := newTrackerOnChain(t, chain)
	got := chain.fetch(t, tr, extra.Timepoints)
	require.Equal(t, now, got[headIndex].BlockTimestamp)
	require.Equal(t, onchain[windowStart].TickCumulative, got[windowStart].TickCumulative)
	require.Contains(t, chain.takeRequested(), headIndex)

	extra.Timepoints = got
	require.Equal(t, onchainFee, quote(extra), "fixed tracker output must match the on-chain fee")

	// the same pool without any adaptive fee keeps lastFee (725), the on-chain fee
	noDynFee := extra
	noDynFee.DynamicFee = nil
	require.Equal(t, onchainFee, quote(noDynFee))
}

// stalePoolFixture is thena-fusion-v3 pool 0x7491c04d...fb7dcc1 as stored by pool-service, with the stale timepoint
// at the current index 41557 (ts 1780562536, while the plugin's lastTimepointTimestamp is 1790916437).
const stalePoolFixture = `{"address":"0x7491c04dc4575e086a8ee31f7ce1c6d56fb7dcc1","swapFee":725,"exchange":"thena-fusion-v3","type":"algebra-integral","timestamp":1790916437,"reserves":["25595617448839672865479","35844268369299969675617"],"tokens":[{"address":"0x55d398326f99059ff775485246999027b3197955","symbol":"USDT","decimals":18,"swappable":true},{"address":"0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d","symbol":"USDC","decimals":18,"swappable":true}],"extra":"{\"liq\":\"1200337013101616101373234590\",\"gS\":{\"price\":\"79214682192264360843718915027\",\"tick\":-4,\"lF\":725,\"pC\":195,\"cF\":1000,\"un\":true},\"ticks\":[{\"Index\":-3540,\"LiquidityGross\":\"160582453969540405275\",\"LiquidityNet\":\"160582453969540405275\"},{\"Index\":-986,\"LiquidityGross\":\"30979464077477145\",\"LiquidityNet\":\"30979464077477145\"},{\"Index\":-319,\"LiquidityGross\":\"30979464077477145\",\"LiquidityNet\":\"-30979464077477145\"},{\"Index\":-207,\"LiquidityGross\":\"25094507970775923334\",\"LiquidityNet\":\"25094507970775923334\"},{\"Index\":-97,\"LiquidityGross\":\"529580698903303070432\",\"LiquidityNet\":\"529580698903303070432\"},{\"Index\":-60,\"LiquidityGross\":\"3947577744916743530583\",\"LiquidityNet\":\"3947577744916743530583\"},{\"Index\":-11,\"LiquidityGross\":\"192708139288096884217\",\"LiquidityNet\":\"192708139288096884217\"},{\"Index\":-10,\"LiquidityGross\":\"1441372010033440212321\",\"LiquidityNet\":\"1441372010033440212321\"},{\"Index\":-9,\"LiquidityGross\":\"10040161254876749351411\",\"LiquidityNet\":\"10040161254876749351411\"},{\"Index\":-4,\"LiquidityGross\":\"1200320676024806142723857017\",\"LiquidityNet\":\"1200320676024806142723857017\"},{\"Index\":-3,\"LiquidityGross\":\"1200247807444761074611960299\",\"LiquidityNet\":\"-1200247807444761074611960299\"},{\"Index\":-1,\"LiquidityGross\":\"10232869394164846235628\",\"LiquidityNet\":\"-10232869394164846235628\"},{\"Index\":1,\"LiquidityGross\":\"4232797084627335447872941\",\"LiquidityNet\":\"4232797084627335447872941\"},{\"Index\":2,\"LiquidityGross\":\"970408054624864614897495\",\"LiquidityNet\":\"966234244780002021189429\"},{\"Index\":3,\"LiquidityGross\":\"968321150965263249405966\",\"LiquidityNet\":\"-968321148439603386680958\"},{\"Index\":5,\"LiquidityGross\":\"72482517836113823822609\",\"LiquidityNet\":\"-72482517836113823822609\"},{\"Index\":6,\"LiquidityGross\":\"386062208954288074109\",\"LiquidityNet\":\"-386062208954288074109\"},{\"Index\":7,\"LiquidityGross\":\"4230710179704904151018908\",\"LiquidityNet\":\"-4230710179704904151018908\"},{\"Index\":8,\"LiquidityGross\":\"1262829931362504\",\"LiquidityNet\":\"-1262829931362504\"},{\"Index\":10,\"LiquidityGross\":\"1441372010033440212321\",\"LiquidityNet\":\"-1441372010033440212321\"},{\"Index\":60,\"LiquidityGross\":\"3947577744916743530583\",\"LiquidityNet\":\"-3947577744916743530583\"},{\"Index\":103,\"LiquidityGross\":\"529580698903303070432\",\"LiquidityNet\":\"-529580698903303070432\"},{\"Index\":193,\"LiquidityGross\":\"25094507970775923334\",\"LiquidityNet\":\"-25094507970775923334\"},{\"Index\":2640,\"LiquidityGross\":\"160582453969540405275\",\"LiquidityNet\":\"-160582453969540405275\"}],\"tS\":1,\"tP\":{\"0\":{\"init\":true,\"ts\":1744705175,\"vo\":\"0\",\"tick\":-2,\"avgT\":-2},\"41557\":{\"init\":true,\"ts\":1780562536,\"cum\":-8527355,\"vo\":\"85242944\",\"tick\":-7,\"avgT\":-9,\"wsI\":39491},\"41558\":{\"init\":true,\"ts\":1780562543,\"cum\":-8527411,\"vo\":\"85242951\",\"tick\":-8,\"avgT\":-9,\"wsI\":39491},\"41559\":{\"init\":true,\"ts\":1780562544,\"cum\":-8527419,\"vo\":\"85242952\",\"tick\":-8,\"avgT\":-9,\"wsI\":39491}},\"vo\":{\"init\":true,\"tpIdx\":41557,\"lastTs\":1790916437},\"dF\":{\"a1\":500,\"a2\":200,\"b1\":360,\"b2\":60000,\"g1\":59,\"g2\":8500,\"bF\":25},\"sF\":{\"0to1fF\":\"79228162514264337593543950336\",\"1to0fF\":\"79228162514264337593543950336\",\"pCF\":1000,\"bF\":3000}}","staticExtra":"{\"poolId\":\"0x7491c04dc4575e086a8ee31f7ce1c6d56fb7dcc1\"}","blockNumber":125230986}`

// onchainTimepointsFixture holds the on-chain timepoints the simulator reads for that pool at index 41557:
// the head, its window start (41256) with neighbours, the oldest ones (41558, 41559) and slot 0.
const onchainTimepointsFixture = `{"0":{"avgT":-6,"cum":-32563279,"init":true,"tick":-6,"ts":1783659516,"vo":"92493043","wsI":65190},"41256":{"avgT":-5,"cum":-72751097,"init":true,"tick":-8,"ts":1790829857,"vo":"3149047994275","wsI":40658},"41257":{"avgT":-5,"cum":-72763841,"init":true,"tick":-8,"ts":1790831450,"vo":"3149048008612","wsI":40658},"41556":{"avgT":-14,"cum":-74028713,"init":true,"tick":-4,"ts":1790916237,"vo":"3936278050862","wsI":41255},"41557":{"avgT":-14,"cum":-74029513,"init":true,"tick":-4,"ts":1790916437,"vo":"3936278070862","wsI":41256},"41558":{"avgT":-9,"cum":-8527411,"init":true,"tick":-8,"ts":1780562543,"vo":"85242951","wsI":39491},"41559":{"avgT":-9,"cum":-8527419,"init":true,"tick":-8,"ts":1780562544,"vo":"85242952","wsI":39491}}`
