package efficientprop

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// forkCheck is one size checked on the fork: the simulator's answer and the door's at the refresh block.
type forkCheck struct {
	Door     string `json:"door"`
	TokenIn  string `json:"tokenIn"`
	AmountIn string `json:"amountIn"`
	Sim      string `json:"sim,omitempty"`   // empty: the simulator quotes nothing
	Quote    string `json:"quote,omitempty"` // empty: the door refuses
	GapBps   int64  `json:"gapBps,omitempty"`
}

// TestForkSimulatorAgainstTheDoor runs against a chain with a deployed front, door factory and opened listing doors
// (a local fork with a listing, or the chain itself). It lists the front's doors, refreshes each, and checks the
// simulator against the door's own quote at the refresh block: at every sampled size, between them, outside them,
// and at a sweep of sizes by the input token's decimals. The simulator must never pay more than the door, nor quote a
// size the door refuses.
//
//	EFFICIENT_PROP_RPC=<rpc> EFFICIENT_PROP_FRONT=<front> EFFICIENT_PROP_FACTORY=<door factory> \
//	  go test ./pkg/liquidity-source/efficient-prop -run TestFork -v
//
// EFFICIENT_PROP_CHAIN sets the chain id (default Base), EFFICIENT_PROP_MULTICALL the Multicall3 address,
// EFFICIENT_PROP_FILL_MODE the front's fill mode (default front), EFFICIENT_PROP_SOURCES the front's approved price
// sources (a comma list; default none, so a listing that reads a source quotes nothing), EFFICIENT_PROP_IMPLEMENTATIONS
// the front's listed implementations (a comma list; default the one its proxy runs at the head), and EFFICIENT_PROP_OUT
// a file the checks are written to. Each pool's meta must name its door's front and listing, and in the front's mode
// the front must name the listing's pool (payPoolOf), so the adapter takes the paid entry whose gas is declared.
//
// EFFICIENT_PROP_STATE names a file of refreshed pools. Where it holds an earlier run's, their simulators are first
// priced against the door now (gapToNow): with no refresh trigger logged since, never over it; after a post or a
// fill, which the event parser maps to the door, the gap is logged. The fresh pools are then written to it.
//
// EFFICIENT_PROP_EXPIRE=1, on a local fork only (it moves the fork's clock and mines): after the checks, each listing's
// fork is moved to its state's ValidUntil, where the door must still fill the ladder's first size and route finding
// must quote no later fill, then a second at a time to the word's end, where the door must refuse it and a new
// refresh must quote nothing.
func TestForkSimulatorAgainstTheDoor(t *testing.T) {
	rpcURL, front, factory := os.Getenv("EFFICIENT_PROP_RPC"), os.Getenv("EFFICIENT_PROP_FRONT"),
		os.Getenv("EFFICIENT_PROP_FACTORY")
	if rpcURL == "" || front == "" || factory == "" {
		t.Skip("EFFICIENT_PROP_RPC, EFFICIENT_PROP_FRONT, EFFICIENT_PROP_FACTORY are " +
			"not all set")
	}
	skipOnABuilderChain(t, rpcURL)
	chain := valueobject.ChainIDBase
	if s := os.Getenv("EFFICIENT_PROP_CHAIN"); s != "" {
		id, err := strconv.ParseUint(s, 10, 64)
		require.NoError(t, err)
		chain = valueobject.ChainID(id)
	}
	mc := multicall3
	if s := os.Getenv("EFFICIENT_PROP_MULTICALL"); s != "" {
		mc = common.HexToAddress(s)
	}
	ctx := context.Background()
	f := &forkRun{t: t, ctx: ctx, client: ethrpc.New(rpcURL).SetMulticallContract(mc)}
	cfg := &Config{ChainID: chain, Fronts: []Front{forkFront(front, factory)}}

	if path := os.Getenv("EFFICIENT_PROP_STATE"); path != "" {
		if b, err := os.ReadFile(path); err == nil {
			var earlier []entity.Pool
			require.NoError(t, json.Unmarshal(b, &earlier))
			parser := NewPoolFactory(cfg, f.client)
			for _, p := range earlier {
				f.gapToNow(p, parser)
			}
		}
	}

	u, err := NewPoolsListUpdater(cfg, f.client)
	require.NoError(t, err)
	pools, _, err := u.GetNewPools(ctx, nil)
	require.NoError(t, err)
	require.NotEmpty(t, pools, "the front has no opened listing door")

	tr, err := NewPoolTracker(cfg, f.client)
	require.NoError(t, err)
	var checks []forkCheck
	for k, p := range pools {
		for _, tok := range p.Tokens {
			var dec uint8
			_, err := f.client.NewRequest().SetContext(ctx).
				AddCall(&ethrpc.Call{ABI: abi.Erc20ABI, Target: tok.Address, Method: "decimals"}, []any{&dec}).Call()
			require.NoError(t, err)
			tok.Decimals = dec
		}
		// the second refresh sizes from the first's ladder, as every refresh after a listing's first does
		for range 2 {
			p, err = tr.GetNewPoolState(ctx, p, pool.GetNewPoolStateParams{})
			require.NoError(t, err)
		}
		pools[k] = p
		var se StaticExtra
		require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &se))
		doorHash, err := newCodeHashes(f.client).door(ctx, cfg.Fronts[0])
		require.NoError(t, err)
		require.Equal(t, p.Address, doorOf(cfg.Fronts[0], doorHash, se.Listing),
			"the parser's door is the factory's doorOf")
		f.checkMeta(p, se)
		e := f.extra(p)
		var tops [2]string
		for dir, l := range e.Ladders {
			if len(l) > 0 {
				tops[dir] = l[len(l)-1][0].Dec() + " -> " + l[len(l)-1][1].Dec()
			}
		}
		t.Logf("door %s at block %d (unix %d): %d and %d points, tops %v, valid until %d, custody %v, tip cap %d wei",
			p.Address, p.BlockNumber, f.blockTime(p.BlockNumber), len(e.Ladders[0]), len(e.Ladders[1]), tops,
			e.ValidUntil, p.Reserves, e.TipCapWei)
		checks = append(checks, f.check(p)...)
	}
	if out := os.Getenv("EFFICIENT_PROP_OUT"); out != "" {
		b, err := json.MarshalIndent(checks, "", " ")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(out, b, 0o644))
	}
	if path := os.Getenv("EFFICIENT_PROP_STATE"); path != "" {
		b, err := json.Marshal(pools)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(path, b, 0o644))
	}
	if os.Getenv("EFFICIENT_PROP_EXPIRE") == "1" {
		for _, p := range pools {
			f.expire(tr, p)
		}
	}
}

// skipOnABuilderChain skips a head-read fork test where the node's chain is a builder chain (Ethereum). These tests
// read a listing at the head and fill or quote it there, which is the storage build's path (Base). On a builder chain
// a listing's word stands only in the block a builder lands its desk's update in, so the head holds no quote, the
// parser maps no lane's post, and a tip cap of 0 means no cap: a listing there is read through /state, which
// TestForkBuilderFillsTheSampledQuote covers, with TestForkBuilderMapsAFillsSwappedToItsDoor for its logs.
func skipOnABuilderChain(t *testing.T, rpcURL string) {
	t.Helper()
	id, err := ethrpc.New(rpcURL).GetETHClient().ChainID(context.Background())
	require.NoError(t, err)
	if builderChains[valueobject.ChainID(id.Uint64())] {
		t.Skipf("chain %d is a builder chain: this test reads the head, the storage build's path; run "+
			"TestForkBuilder there", id)
	}
}

// forkFront is the fork's front config: its fill mode, EFFICIENT_PROP_FILL_MODE (default front), its approved
// sources, EFFICIENT_PROP_SOURCES (a comma list, default none), its implementations, EFFICIENT_PROP_IMPLEMENTATIONS
// (a comma list, default the one the front's proxy runs at the head of EFFICIENT_PROP_RPC), and the chain's
// millisecond-time contract, EFFICIENT_PROP_BASE_TIME (forkBaseTime).
func forkFront(front, factory string) Front {
	bt := forkBaseTime()
	f := Front{Address: front, DoorFactory: factory, FillMode: FillModeFront,
		BaseTime: hexutil.Encode(bt[:])}
	if m := os.Getenv("EFFICIENT_PROP_FILL_MODE"); m != "" {
		f.FillMode = m
	}
	if s := os.Getenv("EFFICIENT_PROP_SOURCES"); s != "" {
		f.ApprovedSources = strings.Split(s, ",")
	}
	if s := os.Getenv("EFFICIENT_PROP_IMPLEMENTATIONS"); s != "" {
		f.Implementations = strings.Split(s, ",")
	} else if impl := forkImplementation(front); impl != (common.Address{}) {
		f.Implementations = []string{hexutil.Encode(impl[:])}
	}
	return f
}

// forkBaseTime is the chain's millisecond-time contract, EFFICIENT_PROP_BASE_TIME, default Base's predeploy.
func forkBaseTime() common.Address {
	if s := os.Getenv("EFFICIENT_PROP_BASE_TIME"); s != "" {
		return common.HexToAddress(s)
	}
	return testBaseTime
}

// forkImplementation is the front proxy's implementation at the head of EFFICIENT_PROP_RPC (its ERC-1967 slot), the
// zero address where it cannot be read.
func forkImplementation(front string) common.Address {
	b, err := ethrpc.New(os.Getenv("EFFICIENT_PROP_RPC")).GetETHClient().StorageAt(context.Background(),
		common.HexToAddress(front), implementationSlot, nil)
	if err != nil || len(b) != common.HashLength {
		return common.Address{}
	}
	return common.BytesToAddress(b)
}

// checkMeta checks the pool's meta against the chain: the door answers its front and listing, and in the front's mode
// the front names the listing's pool (payPoolOf), the paid entry the declared gas was measured on.
func (f *forkRun) checkMeta(p entity.Pool, se StaticExtra) {
	t := f.t
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
	require.NoError(t, err)
	meta := sim.GetMetaInfo("", "").(PoolMeta)
	var doorFront common.Address
	var doorListing *big.Int
	_, err = f.client.NewRequest().SetContext(f.ctx).
		AddCall(&ethrpc.Call{ABI: doorABI, Target: p.Address, Method: "FRONT"}, []any{&doorFront}).
		AddCall(&ethrpc.Call{ABI: doorABI, Target: p.Address, Method: "LISTING"}, []any{&doorListing}).Aggregate()
	require.NoError(t, err)
	require.Equal(t, p.Address, meta.Door)
	require.Equal(t, hexutil.Encode(doorFront[:]), meta.Front, "the meta's front is the door's")
	require.Equal(t, doorListing.Uint64(), uint64(meta.Listing), "the meta's listing is the door's")
	if meta.Mode != modeFront {
		require.Equal(t, p.Address, meta.ApprovalAddress)
		return
	}
	require.Equal(t, meta.Front, meta.ApprovalAddress)
	sel := crypto.Keccak256([]byte("payPoolOf(uint24)"))[:4]
	out, err := f.client.GetETHClient().CallContract(f.ctx, ethereum.CallMsg{To: &doorFront,
		Data: append(sel, common.BigToHash(doorListing).Bytes()...)}, new(big.Int).SetUint64(p.BlockNumber))
	require.NoError(t, err)
	require.Len(t, out, 32)
	payPool := common.BytesToAddress(out)
	require.Equal(t, se.Custody, hexutil.Encode(payPool[:]), "the front pays a quoted listing's pool directly")
	t.Logf("door %s: mode front, data door %s front %s listing %d, payPoolOf %s", p.Address, meta.Door, meta.Front,
		meta.Listing, hexutil.Encode(payPool[:]))
}

// forkRun is one fork test's chain.
type forkRun struct {
	t      *testing.T
	ctx    context.Context
	client *ethrpc.Client
}

func (f *forkRun) extra(p entity.Pool) Extra {
	var e Extra
	require.NoError(f.t, json.Unmarshal([]byte(p.Extra), &e))
	return e
}

func (f *forkRun) blockTime(n uint64) uint64 {
	h, err := f.client.GetETHClient().HeaderByNumber(f.ctx, new(big.Int).SetUint64(n))
	require.NoError(f.t, err)
	return h.Time
}

// doorQuote is the door's quote at block (nil: the latest); ok false where it refuses.
func (f *forkRun) doorQuote(door, tokenIn, tokenOut string, size, block *big.Int) (*big.Int, bool) {
	quote := new(big.Int)
	req := f.client.NewRequest().SetContext(f.ctx)
	if block != nil {
		req.SetBlockNumber(block)
	}
	_, err := req.AddCall(&ethrpc.Call{ABI: doorABI, Target: door, Method: "quote",
		Params: []any{common.HexToAddress(tokenIn), common.HexToAddress(tokenOut), size}}, []any{&quote}).Call()
	return quote, err == nil
}

// sizes are a direction's ladder sizes (forkSizes) and a sweep from 10^(d-4) to 10^(d+3) of its input token.
func sizes(p entity.Pool, e Extra, dir int) []*big.Int {
	out := forkSizes(e.ladders()[dir])
	d := int64(p.Tokens[dir].Decimals)
	for x := max(d-4, 0); x <= d+3; x++ {
		out = append(out, new(big.Int).Exp(big.NewInt(10), big.NewInt(x), nil))
	}
	return out
}

// check prices the pool's simulator against the door at the refresh block, both ways.
func (f *forkRun) check(p entity.Pool) []forkCheck {
	t := f.t
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
	require.NoError(t, err)
	e := f.extra(p)
	block := new(big.Int).SetUint64(p.BlockNumber)
	var checks []forkCheck
	for dir := range 2 {
		tokenIn, tokenOut := p.Tokens[dir].Address, p.Tokens[1-dir].Address
		for _, size := range sizes(p, e, dir) {
			c := forkCheck{Door: p.Address, TokenIn: tokenIn, AmountIn: size.String()}
			res, simErr := calc(sim, tokenIn, tokenOut, size, nil)
			quote, quoted := f.doorQuote(p.Address, tokenIn, tokenOut, size, block)
			if quoted {
				c.Quote = quote.String()
			}
			if simErr == nil {
				out := res.TokenAmountOut.Amount
				c.Sim = out.String()
				require.True(t, quoted, "%s: the simulator quotes %s in where the door refuses", p.Address, size)
				require.LessOrEqual(t, out.Cmp(quote), 0, "%s: at %s in the simulator pays %s, the door %s",
					p.Address, size, out, quote)
				gap := new(big.Int).Sub(quote, out)
				c.GapBps = gap.Mul(gap, big.NewInt(10_000)).Div(gap, quote).Int64()
			}
			checks = append(checks, c)
		}
	}
	return checks
}

// gapToNow prices an earlier refresh's simulator against the door now, at its sampled sizes. Where nothing the event
// parser maps to the door was logged since the state's block (no post, no fill, no settings change) and the state may
// still be quoted (the head's time no later than its ValidUntil), the bound is 0 bp: the state never pays more than
// the door and the door refuses none of its sizes. Where such a log was, the gap is logged: that log is the refresh's
// trigger.
func (f *forkRun) gapToNow(p entity.Pool, parser *EventParser) {
	t := f.t
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
	require.NoError(t, err)
	e := f.extra(p)
	head, err := f.client.GetETHClient().HeaderByNumber(f.ctx, nil)
	require.NoError(t, err)
	var triggers []string
	if head.Number.Uint64() > p.BlockNumber {
		logs, err := f.client.GetETHClient().FilterLogs(f.ctx, ethereum.FilterQuery{
			FromBlock: new(big.Int).SetUint64(p.BlockNumber + 1), ToBlock: head.Number})
		require.NoError(t, err)
		mapped, err := parser.Decode(f.ctx, logs)
		require.NoError(t, err)
		for _, l := range mapped[p.Address] {
			triggers = append(triggers, fmt.Sprintf("%s@%d", l.Topics[0].Hex()[:10], l.BlockNumber))
		}
	}
	bound := len(triggers) == 0 && head.Time <= e.ValidUntil
	for dir := range 2 {
		tokenIn, tokenOut := p.Tokens[dir].Address, p.Tokens[1-dir].Address
		var worst int64
		var refused, n int
		for _, size := range forkSizes(e.Ladders[dir]) {
			res, err := calc(sim, tokenIn, tokenOut, size, nil)
			if err != nil {
				continue
			}
			n++
			quote, ok := f.doorQuote(p.Address, tokenIn, tokenOut, size, nil)
			if !ok {
				refused++
				continue
			}
			over := new(big.Int).Sub(res.TokenAmountOut.Amount, quote)
			worst = max(worst, over.Mul(over, big.NewInt(10_000)).Div(over, quote).Int64())
		}
		f.t.Logf("earlier state of %s (block %d, valid until %d), direction %d: %d sizes, the door now (block %d, "+
			"+%d s) refuses %d, the earlier ladder over the door by up to %d bp; refresh triggers since: %v",
			p.Address, p.BlockNumber, e.ValidUntil, dir, n, head.Number, int64(head.Time)-p.Timestamp, refused, worst,
			triggers)
		if bound {
			require.Zero(t, refused, "%s: no trigger, and the door refuses a size of the state", p.Address)
			require.LessOrEqual(t, worst, int64(0), "%s: no trigger, and the state pays over the door", p.Address)
		}
	}
}

// expire moves a local fork past the pool's ValidUntil (see TestForkSimulatorAgainstTheDoor), from a snapshot it
// reverts to after, so the next listing's state is checked on the clock it was refreshed at.
func (f *forkRun) expire(tr *PoolTracker, p entity.Pool) {
	t := f.t
	rpc := f.client.GetETHClient().Client()
	var snap hexutil.Big
	require.NoError(t, rpc.CallContext(f.ctx, &snap, "evm_snapshot"))
	defer func() {
		var ok bool
		require.NoError(t, rpc.CallContext(f.ctx, &ok, "evm_revert", &snap))
		require.True(t, ok, "the fork reverted to its snapshot")
	}()
	e := f.extra(p)
	dir := 0
	if len(e.Ladders[0]) == 0 {
		dir = 1
	}
	if len(e.Ladders[dir]) == 0 {
		t.Logf("%s quotes nothing: no word to expire", p.Address)
		return
	}
	tokenIn, tokenOut := p.Tokens[dir].Address, p.Tokens[1-dir].Address
	first := e.Ladders[dir][0][0].ToBig()
	head, err := f.client.GetETHClient().HeaderByNumber(f.ctx, nil)
	require.NoError(t, err)
	mineAt := func(ts uint64) {
		require.NoError(t, rpc.CallContext(f.ctx, nil, "evm_setNextBlockTimestamp", ts))
		require.NoError(t, rpc.CallContext(f.ctx, nil, "evm_mine"))
	}
	require.GreaterOrEqual(t, e.ValidUntil, head.Time, "%s: the state is past its word already", p.Address)
	if e.ValidUntil > head.Time {
		mineAt(e.ValidUntil)
	}
	_, ok := f.doorQuote(p.Address, tokenIn, tokenOut, first, nil)
	require.True(t, ok, "%s: the door refuses at the state's last valid second %d", p.Address, e.ValidUntil)
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
	require.NoError(t, err)
	require.False(t, sim.validAt(e.ValidUntil-e.Allowance+1), "route finding quotes a fill landing past the state")

	// on to the word's end, within the front's cap on a word's life
	end := e.ValidUntil + 1
	for ; ; end++ {
		require.LessOrEqual(t, end, e.ValidUntil+32, "%s: the word outlives the front's cap", p.Address)
		mineAt(end)
		if _, ok = f.doorQuote(p.Address, tokenIn, tokenOut, first, nil); !ok {
			break
		}
	}

	p, err = tr.GetNewPoolState(f.ctx, p, pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	after := f.extra(p)
	require.Empty(t, after.Ladders[0], "a ladder from an expired word")
	require.Empty(t, after.Ladders[1], "a ladder from an expired word")
	for _, c := range f.check(p) {
		require.Empty(t, c.Sim, "the simulator quotes %s at %s past the word", c.TokenIn, c.AmountIn)
		require.Empty(t, c.Quote, "the door fills %s at %s past the word", c.TokenIn, c.AmountIn)
	}
	t.Logf("%s: the door fills at the state's ValidUntil %d and refuses from %d; a refresh then has no ladder",
		p.Address, e.ValidUntil, end)
}

// forkSizes are a ladder's points, a third and a half of the way between each two, one under the first and one
// over the last.
func forkSizes(l []Point) []*big.Int {
	var sizes []*big.Int
	for k, p := range l {
		x := p[0].ToBig()
		if k == 0 {
			sizes = append(sizes, new(big.Int).Sub(x, big.NewInt(1)), new(big.Int).Div(x, big.NewInt(2)))
		} else {
			lo := l[k-1][0].ToBig()
			span := new(big.Int).Sub(x, lo)
			sizes = append(sizes,
				new(big.Int).Add(lo, new(big.Int).Div(span, big.NewInt(3))),
				new(big.Int).Add(lo, new(big.Int).Div(span, big.NewInt(2))))
		}
		sizes = append(sizes, x)
	}
	if len(l) > 0 {
		top := l[len(l)-1][0]
		sizes = append(sizes, new(uint256.Int).AddUint64(top, 1).ToBig())
	}
	out := sizes[:0]
	for _, s := range sizes {
		if s.Sign() > 0 {
			out = append(out, s)
		}
	}
	return out
}

// TestForkLadderHoldsOnAnAgingListing, on a local fork only (EFFICIENT_PROP_AGING=1, with the variables of
// TestForkSimulatorAgainstTheDoor; it writes the front's storage): for each listed listing, twice, it sets the
// listing's controls and depth, refreshes it, and prices the state's simulator against the door's quote in every block
// a fill quoted from the state may land in, up to its ValidUntil (an eth_call with a block override on the refresh
// block: on a block-count clock each 200 ms block by its number, so the front's clock reads that block's
// milliseconds): the simulator never pays more. The first setting widens the levels with age (200 ppm/sqrt s, no
// grace) and drifts the bid's depth 20 ppm/s and the ask's 50, so a side paired with the other's depth view shows; the
// second widens them and adds a premium of 2,000 ppm for 2.5 s
// after the maker's post, which ends inside a second.
func TestForkLadderHoldsOnAnAgingListing(t *testing.T) {
	rpcURL, front, factory := os.Getenv("EFFICIENT_PROP_RPC"), os.Getenv("EFFICIENT_PROP_FRONT"),
		os.Getenv("EFFICIENT_PROP_FACTORY")
	if os.Getenv("EFFICIENT_PROP_AGING") != "1" || rpcURL == "" || front == "" || factory == "" {
		t.Skip("EFFICIENT_PROP_AGING=1 and the fork's variables are not all set")
	}
	skipOnABuilderChain(t, rpcURL)
	mc := multicall3
	if s := os.Getenv("EFFICIENT_PROP_MULTICALL"); s != "" {
		mc = common.HexToAddress(s)
	}
	ctx := context.Background()
	f := &forkRun{t: t, ctx: ctx, client: ethrpc.New(rpcURL).SetMulticallContract(mc)}
	cfg := &Config{ChainID: valueobject.ChainIDBase, Fronts: []Front{forkFront(front, factory)}}
	u, err := NewPoolsListUpdater(cfg, f.client)
	require.NoError(t, err)
	pools, _, err := u.GetNewPools(ctx, nil)
	require.NoError(t, err)
	require.NotEmpty(t, pools)
	tr, err := NewPoolTracker(cfg, f.client)
	require.NoError(t, err)
	var clock frontClock
	_, err = f.client.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: frontABI, Target: front, Method: "clock"}, []any{&clock}).Call()
	require.NoError(t, err)
	rpcc := f.client.GetETHClient().Client()
	field := func(w *big.Int, lo, n uint, v int64) *big.Int {
		mask := new(big.Int).Lsh(new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), n), big.NewInt(1)), lo)
		w = new(big.Int).AndNot(w, mask)
		return w.Or(w, new(big.Int).Lsh(big.NewInt(v), lo))
	}
	settings := []struct {
		name                   string
		widen, bidDrift, drift int64
		premium, windowMs      int64
	}{
		{name: "widening 200 ppm/sqrt s, drift bid 20 ask 50 ppm/s", widen: 200, bidDrift: 20, drift: 50},
		{name: "widening 200 ppm/sqrt s, premium 2,000 ppm for 2,500 ms", widen: 200, premium: 2_000, windowMs: 2_500},
	}
	for _, p := range pools {
		var se StaticExtra
		require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &se))
		slot := controlsSlot(se.Listing).Big()
		set := func(slot *big.Int, word *big.Int) {
			require.NoError(t, rpcc.CallContext(ctx, nil, "anvil_setStorageAt", common.HexToAddress(front),
				common.BigToHash(slot), common.BigToHash(word)))
		}
		get := func(slot *big.Int) *big.Int {
			b, err := f.client.GetETHClient().StorageAt(ctx, common.HexToAddress(front), common.BigToHash(slot), nil)
			require.NoError(t, err)
			return new(big.Int).SetBytes(b)
		}
		// the controls word, then the bid's and the ask's depth word 0 (record words 7, 3 and 5), restored at the end
		slots := []*big.Int{slot, new(big.Int).Sub(slot, big.NewInt(4)), new(big.Int).Sub(slot, big.NewInt(2))}
		was := make([]*big.Int, len(slots))
		for k, s := range slots {
			was[k] = get(s)
		}
		for _, tok := range p.Tokens {
			var dec uint8
			_, err := f.client.NewRequest().SetContext(ctx).
				AddCall(&ethrpc.Call{ABI: abi.Erc20ABI, Target: tok.Address, Method: "decimals"}, []any{&dec}).Call()
			require.NoError(t, err)
			tok.Decimals = dec
		}
		for _, st := range settings {
			set(slots[0], field(field(field(field(was[0], 136, 20, st.widen), 156, 20, st.premium), 176, 16,
				st.windowMs), 192, 16, 0))
			set(slots[1], field(was[1], 92, 20, st.bidDrift))
			set(slots[2], field(was[2], 92, 20, st.drift))
			for range 2 {
				p, err = tr.GetNewPoolState(ctx, p, pool.GetNewPoolStateParams{})
				require.NoError(t, err)
			}
			e := f.extra(p)
			if e.Print && len(e.Ladders[0])+len(e.Ladders[1]) == 0 {
				t.Logf("%s reads a price source not approved (EFFICIENT_PROP_SOURCES) or is tied: "+
					"not quoted", p.Address)
				break
			}
			require.NotEmpty(t, e.Ladders[0], "%s: no ladder; post a fresh word first", p.Address)
			last := e.ValidUntil
			// a fresh word outlives the window: the samples were read, and no cap stopped the state short; a print
			// may end sooner
			window := uint64(p.Timestamp) + uint64(maxAge/time.Second) + e.Allowance
			if e.Print {
				require.LessOrEqual(t, last, window, "%s: valid past the sampled window", p.Address)
			} else {
				require.Equal(t, window, last, "%s: valid to the sampled window's end; post a fresh word first",
					p.Address)
			}
			sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
			require.NoError(t, err)
			block := new(big.Int).SetUint64(p.BlockNumber)
			pinned, fromMs := f.pin(clock, p)
			fills := forkFills(pinned, fromMs, last)
			var checked int
			var worst [2]int64 // the largest gap under the door, bp
			for dir := range 2 {
				tokenIn, tokenOut := p.Tokens[dir].Address, p.Tokens[1-dir].Address
				for _, size := range forkSizes(e.Ladders[dir]) {
					res, err := calc(sim, tokenIn, tokenOut, size, nil)
					if err != nil {
						continue
					}
					for _, o := range fills {
						quote, ok := f.doorQuoteAt(p.Address, tokenIn, tokenOut, size, block, o)
						require.True(t, ok, "%s: the door refuses %s in in block %+v", p.Address, size, o)
						require.LessOrEqual(t, res.TokenAmountOut.Amount.Cmp(quote), 0,
							"%s, %s: at %s in the state pays %s, the door in block %+v %s", p.Address, st.name, size,
							res.TokenAmountOut.Amount, o, quote)
						gap := new(big.Int).Sub(quote, res.TokenAmountOut.Amount)
						worst[dir] = max(worst[dir], gap.Mul(gap, big.NewInt(10_000)).Div(gap, quote).Int64())
						checked++
					}
				}
			}
			t.Logf("%s at block %d, %s: %d checks in %d blocks over %d s, the state never above the door, at most "+
				"%d and %d bp under it", p.Address, p.BlockNumber, st.name, checked, len(fills),
				last-uint64(p.Timestamp), worst[0], worst[1])
		}
		for k, s := range slots {
			set(s, was[k])
		}
	}
}

// TestForkSamplesNoPremiumThatEndedBeforeTheBlock, with EFFICIENT_PROP_AGING=1 on a local fork only (it rewrites the
// listing's controls and restores them): a premium of 2,000 ppm whose window ends 100 ms before the moment the
// front's clock reads at the head, in the head's own second, with no widening and no depth drift on either side. No
// fill can land under it, and nothing else ages the levels, so the ladder, refreshed at the head, must pay what the
// door pays there at every point.
func TestForkSamplesNoPremiumThatEndedBeforeTheBlock(t *testing.T) {
	rpcURL, front, factory := os.Getenv("EFFICIENT_PROP_RPC"), os.Getenv("EFFICIENT_PROP_FRONT"),
		os.Getenv("EFFICIENT_PROP_FACTORY")
	if os.Getenv("EFFICIENT_PROP_AGING") != "1" || rpcURL == "" || front == "" || factory == "" {
		t.Skip("EFFICIENT_PROP_AGING=1 and the fork's variables are not all set")
	}
	skipOnABuilderChain(t, rpcURL)
	ctx := context.Background()
	f := &forkRun{t: t, ctx: ctx, client: ethrpc.New(rpcURL).SetMulticallContract(multicall3)}
	cfg := &Config{ChainID: valueobject.ChainIDBase, Fronts: []Front{forkFront(front, factory)}}
	u, err := NewPoolsListUpdater(cfg, f.client)
	require.NoError(t, err)
	pools, _, err := u.GetNewPools(ctx, nil)
	require.NoError(t, err)
	require.NotEmpty(t, pools)
	var clock frontClock
	_, err = f.client.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: frontABI, Target: front, Method: "clock"}, []any{&clock}).Call()
	require.NoError(t, err)
	rpcc := f.client.GetETHClient().Client()
	tr, err := NewPoolTracker(cfg, f.client)
	require.NoError(t, err)
	for _, p := range pools {
		var se StaticExtra
		require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &se))
		slot := common.BigToHash(controlsSlot(se.Listing).Big())
		b, err := f.client.GetETHClient().StorageAt(ctx, common.HexToAddress(front), slot, nil)
		require.NoError(t, err)
		was := new(big.Int).SetBytes(b)
		var view doorViewOut
		_, err = f.client.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{ABI: frontABI, Target: front,
			Method: "doorView", Params: []any{big.NewInt(int64(se.Listing))}}, []any{&view}).Call()
		require.NoError(t, err)
		head, err := f.client.GetETHClient().HeaderByNumber(ctx, nil)
		require.NoError(t, err)
		headMs := f.momentOf(clock, entity.Pool{Timestamp: int64(head.Time), BlockNumber: head.Number.Uint64()})
		makerMs := bits(view.View.Word, 72, 48)
		pend := headMs - 100
		require.Greater(t, pend, makerMs, "post a fresh word first")
		require.Equal(t, headMs/1000, pend/1000, "the head reads under 100 ms into its second; push again")
		mask := func(w *big.Int, lo, n uint, v int64) *big.Int {
			m := new(big.Int).Lsh(new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), n), big.NewInt(1)), lo)
			w = new(big.Int).AndNot(w, m)
			return w.Or(w, new(big.Int).Lsh(big.NewInt(v), lo))
		}
		ctl := mask(mask(mask(mask(was, 136, 20, 0), 156, 20, 2_000), 176, 16, int64(pend-makerMs)), 192, 16, 0)
		require.NoError(t, rpcc.CallContext(ctx, nil, "anvil_setStorageAt", common.HexToAddress(front), slot,
			common.BigToHash(ctl)))
		t.Cleanup(func() {
			require.NoError(t, rpcc.CallContext(ctx, nil, "anvil_setStorageAt", common.HexToAddress(front), slot,
				common.BigToHash(was)))
		})
		// the bid's and the ask's depth word 0 (record words 3 and 5) with no drift (bits 92-111)
		for _, w := range []int64{3, 5} {
			depth := recordSlot(se.Listing, w)
			b, err := f.client.GetETHClient().StorageAt(ctx, common.HexToAddress(front), depth, nil)
			require.NoError(t, err)
			d := new(big.Int).SetBytes(b)
			require.NoError(t, rpcc.CallContext(ctx, nil, "anvil_setStorageAt", common.HexToAddress(front), depth,
				common.BigToHash(mask(d, 92, 20, 0))))
			t.Cleanup(func() {
				require.NoError(t, rpcc.CallContext(ctx, nil, "anvil_setStorageAt", common.HexToAddress(front), depth,
					common.BigToHash(d)))
			})
		}
		for _, tok := range p.Tokens {
			var dec uint8
			_, err := f.client.NewRequest().SetContext(ctx).
				AddCall(&ethrpc.Call{ABI: abi.Erc20ABI, Target: tok.Address, Method: "decimals"}, []any{&dec}).Call()
			require.NoError(t, err)
			tok.Decimals = dec
		}
		p, err = tr.GetNewPoolState(ctx, p, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
		require.Equal(t, head.Number.Uint64(), p.BlockNumber, "refreshed at the head")
		e := f.extra(p)
		if e.Print && len(e.Ladders[0])+len(e.Ladders[1]) == 0 {
			t.Logf("%s reads a price source not approved (EFFICIENT_PROP_SOURCES) or is tied: "+
				"not quoted", p.Address)
			continue
		}
		require.NotEmpty(t, e.Ladders[0])
		block := new(big.Int).SetUint64(p.BlockNumber)
		var points int
		for dir := range 2 {
			for _, pt := range e.Ladders[dir] {
				q, ok := f.doorQuoteAt(p.Address, p.Tokens[dir].Address, p.Tokens[1-dir].Address, pt[0].ToBig(), block,
					override{})
				require.True(t, ok)
				require.Equal(t, q.String(), pt[1].Dec(), "direction %d at %s: the ladder against the door at the "+
					"head, the premium over", dir, pt[0])
				points++
			}
		}
		t.Logf("%s at block %d: the head reads %d ms, the premium ended at %d ms; all %d points pay the door's quote "+
			"at the head", p.Address, p.BlockNumber, headMs, pend, points)
	}
}

// forkFills are the blocks a fill of a state may land in, from the state's own moment fromMs (pin) up to the second
// until, each as a block override of the refresh block: the refresh block itself (no override), then on a block-count
// clock pinned on its count every block whose milliseconds from the clock's anchor fall in [fromMs, until], by its time
// and number; pinned on BaseTime, a second or more off the count, every 200 ms from BaseTime's reading and the last
// millisecond of every second, by its time, number and BaseTime's sub-second (frontClock.place); pinned on neither, no
// later block; and on a seconds clock every later second.
func forkFills(c frontClock, fromMs, until uint64) []override {
	fills := []override{{}}
	if c.ClockMode.Uint64() != clockBlocks {
		for at := fromMs/1000 + 1; at <= until; at++ {
			fills = append(fills, override{time: at})
		}
		return fills
	}
	switch c.reads {
	case readsSecond:
		return fills
	case readsBaseTime:
		var moments []uint64
		for ms := fromMs + blockMs; ms/1000 <= until; ms += blockMs {
			moments = append(moments, ms)
		}
		for sec := fromMs / 1000; sec <= until; sec++ {
			if ms := sec*1000 + 999; ms > fromMs && ms%blockMs != fromMs%blockMs {
				moments = append(moments, ms)
			}
		}
		for _, ms := range moments {
			if o, ok := c.place(ms); ok {
				fills = append(fills, o)
			}
		}
		return fills
	}
	ms0, block0 := c.ClockMs.Uint64(), c.ClockBlock.Uint64()
	for k := (fromMs - ms0 + blockMs - 1) / blockMs; ; k++ {
		ms := ms0 + k*blockMs
		if ms/1000 > until {
			return fills
		}
		fills = append(fills, override{time: ms / 1000, number: block0 + k})
	}
}

// momentOf is the front's clock in the state's own block, ms (frontClock.moment, with BaseTime read at that block).
func (f *forkRun) momentOf(c frontClock, p entity.Pool) uint64 {
	_, moment := f.pin(c, p)
	return moment
}

// pin is the front's clock pinned at the state's own block (frontClock.pin, with BaseTime read at that block) and the
// moment it reads there, ms; the state's block must be one whose clock never steps back.
func (f *forkRun) pin(c frontClock, p entity.Pool) (frontClock, uint64) {
	var ms *big.Int
	c.baseTime = forkBaseTime()
	_, _ = f.client.NewRequest().SetContext(f.ctx).SetBlockNumber(new(big.Int).SetUint64(p.BlockNumber)).
		AddCall(&ethrpc.Call{ABI: baseTimeABI, Target: c.baseTime.Hex(), Method: "timestampMs"}, []any{&ms}).
		TryAggregate()
	require.False(f.t, c.pin(uint64(p.Timestamp), p.BlockNumber, ms), "the front's clock may step back at %d",
		p.BlockNumber)
	return c, c.moment(uint64(p.Timestamp), p.BlockNumber, ms)
}

// doorQuoteAt is the door's quote on block's state in the block o names (an eth_call with the tracker's block and
// state overrides, overrideArgs; none where o is zero); ok false where it refuses.
func (f *forkRun) doorQuoteAt(door, tokenIn, tokenOut string, size, block *big.Int, o override) (*big.Int, bool) {
	data, err := doorABI.Pack("quote", common.HexToAddress(tokenIn), common.HexToAddress(tokenOut), size)
	require.NoError(f.t, err)
	args := []any{map[string]any{"to": common.HexToAddress(door), "data": hexutil.Bytes(data)}, hexutil.EncodeBig(block)}
	args = append(args, overrideArgs(o)...)
	var ret hexutil.Bytes
	if err := f.client.GetETHClient().Client().CallContext(f.ctx, &ret, "eth_call", args...); err != nil {
		return nil, false
	}
	out := unpackUint(&doorABI, "quote", ret)
	return out, out != nil
}

// TestForkQuotesNothingOnAZeroTipCap, with EFFICIENT_PROP_AGING=1 on a local fork only (it sends a transaction as each
// listing's pool owner, then reverts the fork to a snapshot): on the front's storage build, a listing whose operator
// sets its taker tip cap to 0 fills nothing. The door then refuses the sizes it filled, and the lister and the tracker
// leave the listing unquotable: no ladder in either direction, no valid-until, no reserves, and a simulator that
// quotes no size.
func TestForkQuotesNothingOnAZeroTipCap(t *testing.T) {
	rpcURL, front, factory := os.Getenv("EFFICIENT_PROP_RPC"), os.Getenv("EFFICIENT_PROP_FRONT"),
		os.Getenv("EFFICIENT_PROP_FACTORY")
	if os.Getenv("EFFICIENT_PROP_AGING") != "1" || rpcURL == "" || front == "" || factory == "" {
		t.Skip("EFFICIENT_PROP_AGING=1 and the fork's variables are not all set")
	}
	skipOnABuilderChain(t, rpcURL)
	ctx := context.Background()
	f := &forkRun{t: t, ctx: ctx, client: ethrpc.New(rpcURL).SetMulticallContract(multicall3)}
	cfg := &Config{ChainID: valueobject.ChainIDBase, Fronts: []Front{forkFront(front, factory)}}
	u, err := NewPoolsListUpdater(cfg, f.client)
	require.NoError(t, err)
	pools, _, err := u.GetNewPools(ctx, nil)
	require.NoError(t, err)
	require.NotEmpty(t, pools)
	tr, err := NewPoolTracker(cfg, f.client)
	require.NoError(t, err)
	eth := f.client.GetETHClient()
	rpcc := eth.Client()
	var snap hexutil.Big
	require.NoError(t, rpcc.CallContext(ctx, &snap, "evm_snapshot"))
	t.Cleanup(func() {
		var ok bool
		require.NoError(t, rpcc.CallContext(ctx, &ok, "evm_revert", &snap))
		require.True(t, ok, "the fork reverted to its snapshot")
	})
	address := func(to common.Address, sig string) common.Address {
		out, err := eth.CallContract(ctx, ethereum.CallMsg{To: &to, Data: crypto.Keccak256([]byte(sig))[:4]}, nil)
		require.NoError(t, err)
		require.Len(t, out, 32)
		return common.BytesToAddress(out)
	}
	for _, p := range pools {
		var se StaticExtra
		require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &se))
		for _, tok := range p.Tokens {
			var dec uint8
			_, err := f.client.NewRequest().SetContext(ctx).
				AddCall(&ethrpc.Call{ABI: abi.Erc20ABI, Target: tok.Address, Method: "decimals"}, []any{&dec}).Call()
			require.NoError(t, err)
			tok.Decimals = dec
		}
		p, err = tr.GetNewPoolState(ctx, p, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
		was := f.extra(p)
		if was.Print && len(was.Ladders[0])+len(was.Ladders[1]) == 0 {
			t.Logf("%s reads a price source not approved (EFFICIENT_PROP_SOURCES) or is tied: "+
				"not quoted", p.Address)
			continue
		}
		require.NotZero(t, was.TipCapWei, "%s: a storage build's listing that fills has a cap", p.Address)
		require.NotEmpty(t, was.Ladders[0], "%s: no ladder; post a fresh word first", p.Address)

		// the operator's own setTipCap(listing, 0): on a storage build the tightest, so it applies at once
		owner := address(address(common.HexToAddress(se.Custody), "core()"), "owner()")
		data := append(crypto.Keccak256([]byte("setTipCap(uint24,uint16)"))[:4],
			append(common.BigToHash(big.NewInt(int64(se.Listing))).Bytes(), make([]byte, 32)...)...)
		require.NoError(t, rpcc.CallContext(ctx, nil, "anvil_impersonateAccount", owner))
		var hash common.Hash
		require.NoError(t, rpcc.CallContext(ctx, &hash, "eth_sendTransaction", map[string]any{"from": owner,
			"to": common.HexToAddress(front), "data": hexutil.Bytes(data)}))
		require.NoError(t, rpcc.CallContext(ctx, nil, "anvil_stopImpersonatingAccount", owner))
		var receipt *types.Receipt
		for range 50 { // anvil mines the transaction just after it answers
			if receipt, err = eth.TransactionReceipt(ctx, hash); err == nil {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		require.NoError(t, err)
		require.Equal(t, uint64(1), receipt.Status, "%s: setTipCap(%d, 0)", p.Address, se.Listing)

		for dir := range 2 {
			for _, pt := range was.Ladders[dir] {
				_, ok := f.doorQuote(p.Address, p.Tokens[dir].Address, p.Tokens[1-dir].Address, pt[0].ToBig(), nil)
				require.False(t, ok, "%s: the door fills %s in direction %d at a zero cap", p.Address, pt[0], dir)
			}
		}

		// the lister may keep it (a cap of 0 passes any floor), but the tracker quotes no size
		listed, _, err := u.GetNewPools(ctx, nil)
		require.NoError(t, err)
		var kept bool
		for _, q := range listed {
			kept = kept || q.Address == p.Address
		}
		p, err = tr.GetNewPoolState(ctx, p, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
		e := f.extra(p)
		require.Zero(t, e.TipCapWei)
		require.Empty(t, e.Ladders[0], "%s: a ladder at a zero cap", p.Address)
		require.Empty(t, e.Ladders[1], "%s: a ladder at a zero cap", p.Address)
		require.Zero(t, e.ValidUntil)
		require.Equal(t, entity.PoolReserves{"0", "0"}, p.Reserves)
		sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
		require.NoError(t, err)
		var refused int
		for dir := range 2 {
			for _, pt := range was.Ladders[dir] {
				_, err := calc(sim, p.Tokens[dir].Address, p.Tokens[1-dir].Address, pt[0].ToBig(), nil)
				require.Error(t, err, "%s: the simulator quotes %s in direction %d at a zero cap", p.Address, pt[0],
					dir)
				refused++
			}
		}
		t.Logf("%s, listing %d at block %d: tip cap 0 by its owner's setTipCap, listed %v; the door refuses and the "+
			"simulator quotes none of the %d sizes the state quoted at a cap of %d wei", p.Address, se.Listing,
			p.BlockNumber, kept, refused, was.TipCapWei)
	}
}

// TestForkServesAPrintListingOnlyOnApprovedSources, with the variables of TestForkSimulatorAgainstTheDoor and
// EFFICIENT_PROP_SOURCES the sources to approve: each listed listing is refreshed with no source approved and with
// those named. A listing bound to no source (record word 10's primary id 0) is quoted either way. A listing bound to
// one is quoted only where the front's entry of every source it is bound to (primary and backup) names an approved
// contract, and its simulator then never pays more than the door at the refresh block (check); one reading a guarded
// entry now (its record word 2) is charged the guarded gas. A listing bound to a guarded entry is not quoted, approved
// or not, where the front's block-count clock reads BaseTime at the state's block (a fork whose count has left it). A
// tied listing is never quoted. The binding is read here from the front's storage by this test's own slots.
func TestForkServesAPrintListingOnlyOnApprovedSources(t *testing.T) {
	rpcURL, front, factory := os.Getenv("EFFICIENT_PROP_RPC"), os.Getenv("EFFICIENT_PROP_FRONT"),
		os.Getenv("EFFICIENT_PROP_FACTORY")
	if rpcURL == "" || front == "" || factory == "" {
		t.Skip("the fork's variables are not all set")
	}
	skipOnABuilderChain(t, rpcURL)
	ctx := context.Background()
	f := &forkRun{t: t, ctx: ctx, client: ethrpc.New(rpcURL).SetMulticallContract(multicall3)}
	named := forkFront(front, factory)
	none := named
	none.ApprovedSources = nil
	trackers := make([]*PoolTracker, 2)
	var pools []entity.Pool
	for k, fr := range []Front{none, named} {
		cfg := &Config{ChainID: valueobject.ChainIDBase, Fronts: []Front{fr}}
		tr, err := NewPoolTracker(cfg, f.client)
		require.NoError(t, err)
		trackers[k] = tr
		if k == 0 {
			u, err := NewPoolsListUpdater(cfg, f.client)
			require.NoError(t, err)
			pools, _, err = u.GetNewPools(ctx, nil)
			require.NoError(t, err)
			require.NotEmpty(t, pools)
		}
	}
	approves := map[common.Address]bool{}
	for _, s := range named.ApprovedSources {
		approves[common.HexToAddress(s)] = true
	}
	word := func(slot common.Hash) *big.Int {
		b, err := f.client.GetETHClient().StorageAt(ctx, common.HexToAddress(front), slot, nil)
		require.NoError(t, err)
		return new(big.Int).SetBytes(b)
	}
	var clock frontClock
	_, err := f.client.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: frontABI, Target: front, Method: "clock"}, []any{&clock}).Call()
	require.NoError(t, err)
	// readsBaseTimeAt is whether the front's block-count clock reads BaseTime at the state's block
	readsBaseTimeAt := func(q entity.Pool) bool {
		if clock.ClockMode.Uint64() != clockBlocks {
			return false
		}
		c, _ := f.pin(clock, q)
		return c.reads == readsBaseTime
	}
	var served, refused, unbound, guarded, tiedN, offCount int
	for _, p := range pools {
		var se StaticExtra
		require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &se))
		for _, tok := range p.Tokens {
			var dec uint8
			_, err := f.client.NewRequest().SetContext(ctx).
				AddCall(&ethrpc.Call{ABI: abi.Erc20ABI, Target: tok.Address, Method: "decimals"}, []any{&dec}).Call()
			require.NoError(t, err)
			tok.Decimals = dec
		}
		var view doorViewOut
		_, err := f.client.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{ABI: frontABI, Target: front,
			Method: "doorView", Params: []any{big.NewInt(int64(se.Listing))}}, []any{&view}).Call()
		require.NoError(t, err)
		ids := word(recordSlot(se.Listing, 10))
		var bound []common.Address
		// a tied listing is never quoted, its sources approved or not; a guarded entry (bit 247) is quoted where approved
		// and the front's clock reads its count
		tied := (view.View.Scopes>>1)&15 == tiedScopes
		var guardedEntry bool
		for _, id := range []uint64{bits(ids, 0, 32), bits(ids, 32, 32)} {
			if id != 0 {
				e := word(sourceEntrySlot(uint32(id)))
				bound = append(bound, common.BigToAddress(e))
				guardedEntry = guardedEntry || e.Bit(entryGuardBit) == 1
			}
		}
		reads := printed(view.View.Scopes) && len(bound) > 0
		approved := reads
		for _, src := range bound {
			approved = approved && approves[src]
		}
		var states [2]entity.Pool
		for k, tr := range trackers {
			q := p
			for range 2 {
				q, err = tr.GetNewPoolState(ctx, q, pool.GetNewPoolStateParams{})
				require.NoError(t, err)
			}
			states[k] = q
		}
		quotes := func(q entity.Pool) bool {
			e := f.extra(q)
			return len(e.Ladders[0])+len(e.Ladders[1]) > 0
		}
		switch {
		case reads && tied:
			tiedN++
			for k := range states {
				require.False(t, quotes(states[k]), "%s, listing %d is quoted though tied", p.Address, se.Listing)
				require.Equal(t, entity.PoolReserves{"0", "0"}, states[k].Reserves)
			}
		case !reads:
			unbound++
			require.True(t, quotes(states[0]), "%s, listing %d reads no source and is not quoted", p.Address,
				se.Listing)
			require.True(t, quotes(states[1]), "%s, listing %d reads no source and is not quoted", p.Address,
				se.Listing)
		case approved && guardedEntry && readsBaseTimeAt(states[1]):
			// a guarded hold is not sampled where a block may store any sub-second
			offCount++
			for k := range states {
				require.False(t, quotes(states[k]), "%s, listing %d is quoted on a guarded entry where the front's clock "+
					"reads BaseTime", p.Address, se.Listing)
				require.Equal(t, entity.PoolReserves{"0", "0"}, states[k].Reserves)
			}
		case approved:
			served++
			require.False(t, quotes(states[0]), "%s, listing %d is quoted with no source approved", p.Address,
				se.Listing)
			require.True(t, quotes(states[1]), "%s, listing %d is not quoted on its approved sources %v", p.Address,
				se.Listing, bound)
			require.True(t, f.extra(states[1]).Print, "%s: a listing that reads a source is charged the print gas",
				p.Address)
			// the gas class follows the source the listing reads now: its record word 2's guard
			liveGuarded := guardedEntry
			if live := word(recordSlot(se.Listing, 2)); guardedEntry && live.Sign() != 0 {
				liveGuarded = live.Bit(entryGuardBit) == 1
			}
			require.Equal(t, liveGuarded, f.extra(states[1]).Guarded, "%s: the guarded gas where the source it reads "+
				"now is guarded", p.Address)
			if guardedEntry {
				guarded++
			}
			f.check(states[1])
		default:
			refused++
			for k := range states {
				require.False(t, quotes(states[k]), "%s, listing %d is quoted on sources %v not all approved",
					p.Address, se.Listing, bound)
				require.Equal(t, entity.PoolReserves{"0", "0"}, states[k].Reserves)
			}
		}
		t.Logf("%s, listing %d, scopes %d, bound to %v, guarded %v, tied %v: quoted with none approved %v, with %v "+
			"approved %v", p.Address, se.Listing, view.View.Scopes, bound, guardedEntry, tied, quotes(states[0]),
			named.ApprovedSources, quotes(states[1]))
	}
	t.Logf("%d listings read no source and are quoted, %d read only approved sources and are quoted once named (%d of "+
		"them guarded), %d read only approved sources, one guarded, and are not quoted where the front's clock reads "+
		"BaseTime, %d read a source not approved and are not quoted, %d are tied and are not quoted", unbound, served,
		guarded, offCount, refused, tiedN)
}

// TestForkMapsALanesPostToItsListings, with the variables of TestForkServesAPrintListingOnlyOnApprovedSources, takes
// the latest block within the last 20 that holds a post (Posted) of a source EFFICIENT_PROP_SOURCES approves, hands
// that block's logs to the event parser, and refreshes every listed listing: a listing that reads a print and is
// quoted, so bound only to approved sources, gets the post of each approved lane it is bound to; a listing that reads
// no print gets none. A post of a source not approved maps to nothing.
func TestForkMapsALanesPostToItsListings(t *testing.T) {
	rpcURL, front, factory := os.Getenv("EFFICIENT_PROP_RPC"), os.Getenv("EFFICIENT_PROP_FRONT"),
		os.Getenv("EFFICIENT_PROP_FACTORY")
	if rpcURL == "" || front == "" || factory == "" || os.Getenv("EFFICIENT_PROP_SOURCES") == "" {
		t.Skip("the fork's variables and EFFICIENT_PROP_SOURCES are not all set")
	}
	skipOnABuilderChain(t, rpcURL)
	ctx := context.Background()
	f := &forkRun{t: t, ctx: ctx, client: ethrpc.New(rpcURL).SetMulticallContract(multicall3)}
	cfg := &Config{ChainID: valueobject.ChainIDBase, Fronts: []Front{forkFront(front, factory)}}
	eth := f.client.GetETHClient()
	head, err := eth.HeaderByNumber(ctx, nil)
	require.NoError(t, err)
	var lanes []common.Address
	for _, s := range cfg.Fronts[0].ApprovedSources {
		lanes = append(lanes, common.HexToAddress(s))
	}
	from := head.Number.Uint64() - min(head.Number.Uint64(), 20)
	posts, err := eth.FilterLogs(ctx, ethereum.FilterQuery{FromBlock: new(big.Int).SetUint64(from),
		ToBlock: head.Number, Topics: [][]common.Hash{{topicPosted}}})
	require.NoError(t, err)
	var at uint64
	for _, l := range posts {
		if slices.Contains(lanes, l.Address) && new(big.Int).SetBytes(l.Data).Sign() != 0 {
			at = max(at, l.BlockNumber)
		}
	}
	require.NotZero(t, at, "no post of an approved source in blocks %d to %d", from, head.Number)
	logs, err := eth.FilterLogs(ctx, ethereum.FilterQuery{FromBlock: new(big.Int).SetUint64(at),
		ToBlock: new(big.Int).SetUint64(at)})
	require.NoError(t, err)
	mapped, err := NewPoolFactory(cfg, f.client).Decode(ctx, logs)
	require.NoError(t, err)
	postsOf := func(door string) (from []common.Address) {
		for _, l := range mapped[door] {
			if l.Topics[0] == topicPosted {
				require.Contains(t, lanes, l.Address, "%s: a post of a source not approved mapped", door)
				from = append(from, l.Address)
			}
		}
		return from
	}
	u, err := NewPoolsListUpdater(cfg, f.client)
	require.NoError(t, err)
	pools, _, err := u.GetNewPools(ctx, nil)
	require.NoError(t, err)
	tr, err := NewPoolTracker(cfg, f.client)
	require.NoError(t, err)
	var reached int
	for _, p := range pools {
		var se StaticExtra
		require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &se))
		for _, tok := range p.Tokens {
			tok.Decimals = 18
		}
		q, err := tr.GetNewPoolState(ctx, p, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
		e := f.extra(q)
		quoted := len(e.Ladders[0])+len(e.Ladders[1]) > 0
		got := postsOf(p.Address)
		switch {
		case !e.Print:
			require.Empty(t, got, "%s, listing %d reads no print and a lane's post maps to it", p.Address, se.Listing)
		case quoted:
			require.NotEmpty(t, got, "%s, listing %d reads a print on approved sources and no post maps to it",
				p.Address, se.Listing)
			reached++
		}
		t.Logf("%s, listing %d, print %v, quoted %v: posts of %v in block %d map to it", p.Address, se.Listing,
			e.Print, quoted, got, at)
	}
	require.NotZero(t, reached, "no quoted print listing")
}

// TestForkQuotesNothingOffTheImplementationList refreshes every listing of the front twice: with its implementations
// listing the one its proxy runs, where a listing that reads no source is quoted, and listing another, where none is.
func TestForkQuotesNothingOffTheImplementationList(t *testing.T) {
	rpcURL, front, factory := os.Getenv("EFFICIENT_PROP_RPC"), os.Getenv("EFFICIENT_PROP_FRONT"),
		os.Getenv("EFFICIENT_PROP_FACTORY")
	if rpcURL == "" || front == "" || factory == "" {
		t.Skip("the fork's variables are not all set")
	}
	skipOnABuilderChain(t, rpcURL)
	ctx := context.Background()
	f := &forkRun{t: t, ctx: ctx, client: ethrpc.New(rpcURL).SetMulticallContract(multicall3)}
	live := forkImplementation(front)
	require.NotEqual(t, common.Address{}, live, "the front's proxy names no implementation")
	listed := forkFront(front, factory)
	listed.Implementations = []string{hexutil.Encode(live[:])}
	other := listed
	other.Implementations = []string{"0x000000000000000000000000000000000000dead"}
	u, err := NewPoolsListUpdater(&Config{ChainID: valueobject.ChainIDBase, Fronts: []Front{listed}}, f.client)
	require.NoError(t, err)
	pools, _, err := u.GetNewPools(ctx, nil)
	require.NoError(t, err)
	require.NotEmpty(t, pools)
	var quoted int
	for _, fr := range []Front{listed, other} {
		tr, err := NewPoolTracker(&Config{ChainID: valueobject.ChainIDBase, Fronts: []Front{fr}}, f.client)
		require.NoError(t, err)
		for _, p := range pools {
			for _, tok := range p.Tokens {
				var dec uint8
				_, err := f.client.NewRequest().SetContext(ctx).
					AddCall(&ethrpc.Call{ABI: abi.Erc20ABI, Target: tok.Address, Method: "decimals"}, []any{&dec}).Call()
				require.NoError(t, err)
				tok.Decimals = dec
			}
			q, err := tr.GetNewPoolState(ctx, p, pool.GetNewPoolStateParams{})
			require.NoError(t, err)
			e := f.extra(q)
			n := len(e.Ladders[0]) + len(e.Ladders[1])
			if fr.Implementations[0] != listed.Implementations[0] {
				require.Zero(t, n, "%s is quoted off the implementation list", p.Address)
				require.Equal(t, entity.PoolReserves{"0", "0"}, q.Reserves)
				continue
			}
			if !e.Print {
				require.NotZero(t, n, "%s reads no source and is not quoted on its listed implementation", p.Address)
				quoted++
			}
		}
	}
	t.Logf("implementation %s: %d listings that read no source quoted on it, none on another", live, quoted)
}

// TestForkSidesFollowThePairWord, with EFFICIENT_PROP_AGING=1 on a local fork only (it rewrites each listing's depth
// words and restores them): for each listing and each side in turn, that side's depth is set to end a few seconds after
// the head while the other's runs on, and the listing refreshed. The door must fill the state's first size of every
// direction it quotes up to the state's ValidUntil (each second's last moment, placed as the tracker places it), and
// the direction that sells into the shortened side must be refused within a second of it. Both directions must keep a
// ladder: a side paired with the other side's depth view is sampled at the other's end, past its own depth, where the
// door refuses it, and is left with none. The direction is found by the door, not by the pair word.
func TestForkSidesFollowThePairWord(t *testing.T) {
	rpcURL, front, factory := os.Getenv("EFFICIENT_PROP_RPC"), os.Getenv("EFFICIENT_PROP_FRONT"),
		os.Getenv("EFFICIENT_PROP_FACTORY")
	if os.Getenv("EFFICIENT_PROP_AGING") != "1" || rpcURL == "" || front == "" || factory == "" {
		t.Skip("EFFICIENT_PROP_AGING=1 and the fork's variables are not all set")
	}
	skipOnABuilderChain(t, rpcURL)
	ctx := context.Background()
	f := &forkRun{t: t, ctx: ctx, client: ethrpc.New(rpcURL).SetMulticallContract(multicall3)}
	cfg := &Config{ChainID: valueobject.ChainIDBase, Fronts: []Front{forkFront(front, factory)}}
	u, err := NewPoolsListUpdater(cfg, f.client)
	require.NoError(t, err)
	pools, _, err := u.GetNewPools(ctx, nil)
	require.NoError(t, err)
	require.NotEmpty(t, pools)
	tr, err := NewPoolTracker(cfg, f.client)
	require.NoError(t, err)
	var clock frontClock
	_, err = f.client.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: frontABI, Target: front, Method: "clock"}, []any{&clock}).Call()
	require.NoError(t, err)
	rpcc := f.client.GetETHClient().Client()
	set := func(slot common.Hash, w *big.Int) {
		require.NoError(t, rpcc.CallContext(ctx, nil, "anvil_setStorageAt", common.HexToAddress(front), slot,
			common.BigToHash(w)))
	}
	get := func(slot common.Hash) *big.Int {
		b, err := f.client.GetETHClient().StorageAt(ctx, common.HexToAddress(front), slot, nil)
		require.NoError(t, err)
		return new(big.Int).SetBytes(b)
	}
	head, err := f.client.GetETHClient().HeaderByNumber(ctx, nil)
	require.NoError(t, err)
	const short = 3 // the shortened side's depth ends this many seconds after the head
pools:
	for _, p := range pools {
		var se StaticExtra
		require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &se))
		for _, tok := range p.Tokens {
			var dec uint8
			_, err := f.client.NewRequest().SetContext(ctx).
				AddCall(&ethrpc.Call{ABI: abi.Erc20ABI, Target: tok.Address, Method: "decimals"}, []any{&dec}).Call()
			require.NoError(t, err)
			tok.Decimals = dec
		}
		// the bid's and the ask's depth word 0, record words 3 and 5; expiry bits 60-91, s
		sides := map[string]common.Hash{"bid": recordSlot(se.Listing, 3),
			"ask": recordSlot(se.Listing, 5)}
		was := map[string]*big.Int{"bid": get(sides["bid"]), "ask": get(sides["ask"])}
		t.Cleanup(func() {
			for side, slot := range sides {
				set(slot, was[side])
			}
		})
		refusedBy := map[string]int{}
		for _, side := range []string{"bid", "ask"} {
			w := new(big.Int).Set(was[side])
			mask := new(big.Int).Lsh(big.NewInt(1<<32-1), 60)
			w.AndNot(w, mask).Or(w, new(big.Int).Lsh(new(big.Int).SetUint64(head.Time+short), 60))
			set(sides[side], w)
			q := p
			for range 2 {
				q, err = tr.GetNewPoolState(ctx, q, pool.GetNewPoolStateParams{})
				require.NoError(t, err)
			}
			e := f.extra(q)
			if e.Print && len(e.Ladders[0])+len(e.Ladders[1]) == 0 {
				t.Logf("%s reads a price source not approved (EFFICIENT_PROP_SOURCES) or is tied: "+
					"not quoted", p.Address)
				continue pools
			}
			require.NotZero(t, e.ValidUntil, "%s: no state; post a fresh word first", p.Address)
			require.True(t, len(e.Ladders[0]) > 0 && len(e.Ladders[1]) > 0, "%s, %s side short: a direction lost "+
				"its ladder", p.Address, side)
			require.LessOrEqual(t, e.ValidUntil, head.Time+short, "%s, %s side ending at %d: the state is valid to "+
				"%d, past the side's depth", p.Address, side, head.Time+short, e.ValidUntil)
			pinned, _ := f.pin(clock, q)
			block := new(big.Int).SetUint64(q.BlockNumber)
			var stops [2]uint64 // the first second whose last moment the door refuses the direction (0: none)
			for dir, l := range e.Ladders {
				if len(l) == 0 {
					continue
				}
				first := l[0][0].ToBig()
				for sec := uint64(q.Timestamp); sec <= head.Time+short+2; sec++ {
					o, ok := pinned.place(sec*1000 + 999)
					if !ok {
						o = override{time: sec}
					}
					_, filled := f.doorQuoteAt(p.Address, q.Tokens[dir].Address, q.Tokens[1-dir].Address, first,
						block, o)
					if !filled {
						stops[dir] = sec
						break
					}
				}
				require.True(t, stops[dir] == 0 || stops[dir] > e.ValidUntil, "%s, %s side short: direction %d is "+
					"refused at %d, the state valid to %d", p.Address, side, dir, stops[dir], e.ValidUntil)
			}
			// the direction the door refuses first is the one selling into the shortened side
			cut := -1
			for dir, s := range stops {
				if s != 0 && (cut < 0 || s < stops[cut]) {
					cut = dir
				}
			}
			require.GreaterOrEqual(t, cut, 0, "%s: no direction refused after the %s side's depth ended", p.Address,
				side)
			require.LessOrEqual(t, stops[cut], e.ValidUntil+2, "%s, %s side short: the state valid to %d, the door "+
				"refuses from %d", p.Address, side, e.ValidUntil, stops[cut])
			refusedBy[side] = cut
			set(sides[side], was[side])
			t.Logf("%s, listing %d, %s depth ends at %d: valid until %d, the door refuses direction %d (%s in) from "+
				"%d, the other from %d (0: not within the span)", p.Address, se.Listing, side, head.Time+short,
				e.ValidUntil, cut, q.Tokens[cut].Address, stops[cut], stops[1-cut])
		}
		require.NotEqual(t, refusedBy["bid"], refusedBy["ask"], "%s: both sides cut the same direction", p.Address)
	}
}

// TestForkGuardedListingHoldsAcrossItsHold, with EFFICIENT_PROP_HOLD=1 on a local fork on the counted clock only (it
// mines blocks), right after a print post on a guarded source's lane that moves the price, so the lane holds the
// replaced print in the post's block and the next: each listing bound to a guarded entry of an approved source is
// refreshed at the head and its simulator priced against the door's quote in every block of the state's validity, up
// to its ValidUntil, each block mined as the counted clock mines it (its second the count's, BaseTime's stored
// sub-second the count's). The simulator never pays more than the door, nor quotes a size the door refuses, in any
// of them. EFFICIENT_PROP_HOLD_OUT names a file the checks are written to.
//
// EFFICIENT_PROP_HOLD_DRIFT (ppm/s) and EFFICIENT_PROP_HOLD_BUCKET (whole quote tokens/s), from a snapshot reverted
// after each listing, first write the listing's bid side to drift at that rate (its depth word 0, bits 92-111) and its
// outflow bucket to refill at that rate (its controls word, bits 80-103), so the bid is bound by its output: the top it
// fills inside the hold is refused in the first block after the hold ends, and the drift brings it back under the bound
// later. Run one block after a post that moves the price up, that block is the one only the hold's samples reach, and
// a refresh that skips the hold quotes a size the door refuses there.
func TestForkGuardedListingHoldsAcrossItsHold(t *testing.T) {
	rpcURL, front, factory := os.Getenv("EFFICIENT_PROP_RPC"), os.Getenv("EFFICIENT_PROP_FRONT"),
		os.Getenv("EFFICIENT_PROP_FACTORY")
	if os.Getenv("EFFICIENT_PROP_HOLD") != "1" || rpcURL == "" || front == "" || factory == "" {
		t.Skip("EFFICIENT_PROP_HOLD=1 and the fork's variables are not all set")
	}
	skipOnABuilderChain(t, rpcURL)
	ctx := context.Background()
	f := &forkRun{t: t, ctx: ctx, client: ethrpc.New(rpcURL).SetMulticallContract(multicall3)}
	cfg := &Config{ChainID: valueobject.ChainIDBase, Fronts: []Front{forkFront(front, factory)}}
	u, err := NewPoolsListUpdater(cfg, f.client)
	require.NoError(t, err)
	pools, _, err := u.GetNewPools(ctx, nil)
	require.NoError(t, err)
	tr, err := NewPoolTracker(cfg, f.client)
	require.NoError(t, err)
	var clock frontClock
	_, err = f.client.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: frontABI, Target: front, Method: "clock"}, []any{&clock}).Call()
	require.NoError(t, err)
	require.Equal(t, uint64(clockBlocks), clock.ClockMode.Uint64(), "a block-count front")
	ms0, block0 := clock.ClockMs.Uint64(), clock.ClockBlock.Uint64()
	eth := f.client.GetETHClient()
	rpcc := eth.Client()
	// mine is the next block on the counted clock
	mine := func() *types.Header {
		head, err := eth.HeaderByNumber(ctx, nil)
		require.NoError(t, err)
		count := ms0 + (head.Number.Uint64()+1-block0)*blockMs
		require.GreaterOrEqual(t, count/1000, head.Time, "the head is off the counted clock")
		require.NoError(t, rpcc.CallContext(ctx, nil, "anvil_setStorageAt", forkBaseTime(), common.Hash{},
			common.BigToHash(new(big.Int).SetUint64(count%1000))))
		require.NoError(t, rpcc.CallContext(ctx, nil, "evm_setNextBlockTimestamp", count/1000))
		require.NoError(t, rpcc.CallContext(ctx, nil, "evm_mine"))
		next, err := eth.HeaderByNumber(ctx, nil)
		require.NoError(t, err)
		require.Equal(t, head.Number.Uint64()+1, next.Number.Uint64())
		return next
	}
	drift, _ := strconv.ParseInt(os.Getenv("EFFICIENT_PROP_HOLD_DRIFT"), 10, 64)
	bucket, _ := strconv.ParseInt(os.Getenv("EFFICIENT_PROP_HOLD_BUCKET"), 10, 64)
	require.True(t, drift >= 0 && drift < 1<<20 && bucket >= 0 && bucket < 1<<18, "drift %d ppm/s, bucket %d a second",
		drift, bucket)
	frontAddr := common.HexToAddress(front)
	// field is w with its n-bit field at lo set to v
	field := func(w *big.Int, lo, n uint, v int64) *big.Int {
		mask := new(big.Int).Lsh(new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), n), big.NewInt(1)), lo)
		w = new(big.Int).AndNot(w, mask)
		return w.Or(w, new(big.Int).Lsh(big.NewInt(v), lo))
	}
	var checks []forkCheck
	var served int
	for _, p := range pools {
		decimals := make([]uint8, len(p.Tokens))
		for k, tok := range p.Tokens {
			_, err := f.client.NewRequest().SetContext(ctx).
				AddCall(&ethrpc.Call{ABI: abi.Erc20ABI, Target: tok.Address, Method: "decimals"},
					[]any{&decimals[k]}).Call()
			require.NoError(t, err)
			tok.Decimals = decimals[k]
		}
		head, err := eth.HeaderByNumber(ctx, nil)
		require.NoError(t, err)
		q, err := tr.GetNewPoolState(ctx, p, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
		if !f.extra(q).Guarded {
			continue
		}
		var snap hexutil.Big
		require.NoError(t, rpcc.CallContext(ctx, &snap, "evm_snapshot"))
		// a check that fails first still leaves the fork as it found it: the listing's sides and its blocks reverted
		reverted := false
		t.Cleanup(func() {
			var ok bool
			if !reverted && (rpcc.CallContext(ctx, &ok, "evm_revert", &snap) != nil || !ok) {
				t.Errorf("the fork did not revert to its snapshot %s", snap.String())
			}
		})
		if drift != 0 || bucket != 0 {
			var se StaticExtra
			require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &se))
			var view doorViewOut
			_, err := f.client.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{ABI: frontABI, Target: front,
				Method: "doorView", Params: []any{big.NewInt(int64(se.Listing))}}, []any{&view}).Call()
			require.NoError(t, err)
			// the quote token's decimals: token 1's where the base is token 0 (the pair word's bit 8)
			quoteDec := decimals[0]
			if view.View.PairWord != nil && view.View.PairWord.Bit(8) == 1 {
				quoteDec = decimals[1]
			}
			set := func(slot common.Hash, edit func(*big.Int) *big.Int) {
				b, err := eth.StorageAt(ctx, frontAddr, slot, nil)
				require.NoError(t, err)
				require.NoError(t, rpcc.CallContext(ctx, nil, "anvil_setStorageAt", frontAddr, slot,
					common.BigToHash(edit(new(big.Int).SetBytes(b)))))
			}
			if drift != 0 {
				set(recordSlot(se.Listing, 3), func(w *big.Int) *big.Int { return field(w, 92, 20, drift) })
			}
			if bucket != 0 {
				set(controlsSlot(se.Listing), func(w *big.Int) *big.Int {
					return field(w, 80, 24, bucket|int64(quoteDec)<<18)
				})
			}
		}
		// the second refresh sizes from the first's ladder
		p = q
		for range 2 {
			p, err = tr.GetNewPoolState(ctx, p, pool.GetNewPoolStateParams{})
			require.NoError(t, err)
		}
		e := f.extra(p)
		require.True(t, e.Guarded)
		require.Equal(t, head.Number.Uint64(), p.BlockNumber, "refreshed at the post's block")
		require.NotEmpty(t, e.Ladders[0], "%s: a guarded listing on an approved source has no ladder", p.Address)
		require.NotEmpty(t, e.Ladders[1], "%s: a guarded listing on an approved source has no ladder", p.Address)
		served++
		sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
		require.NoError(t, err)
		var blocks, n int
		var worst [2]int64
		for at := head; at.Time <= e.ValidUntil; at = mine() {
			blocks++
			for dir := range 2 {
				tokenIn, tokenOut := p.Tokens[dir].Address, p.Tokens[1-dir].Address
				for _, size := range sizes(p, e, dir) {
					res, simErr := calc(sim, tokenIn, tokenOut, size, nil)
					if simErr != nil {
						continue
					}
					quote, ok := f.doorQuote(p.Address, tokenIn, tokenOut, size, at.Number)
					c := forkCheck{Door: p.Address, TokenIn: tokenIn, AmountIn: size.String(),
						Sim: res.TokenAmountOut.Amount.String()}
					if ok {
						c.Quote = quote.String()
					}
					checks = append(checks, c)
					require.True(t, ok, "%s: the state quotes %s in direction %d, the door refuses it in block %d (+%d "+
						"from the state's)", p.Address, size, dir, at.Number, at.Number.Uint64()-p.BlockNumber)
					require.LessOrEqual(t, res.TokenAmountOut.Amount.Cmp(quote), 0, "%s: at %s in direction %d the "+
						"state pays %s, the door in block %d %s", p.Address, size, dir, res.TokenAmountOut.Amount,
						at.Number, quote)
					gap := new(big.Int).Sub(quote, res.TokenAmountOut.Amount)
					worst[dir] = max(worst[dir], gap.Mul(gap, big.NewInt(10_000)).Div(gap, quote).Int64())
					n++
				}
			}
		}
		var ok bool
		reverted = true
		require.NoError(t, rpcc.CallContext(ctx, &ok, "evm_revert", &snap))
		require.True(t, ok, "the fork reverted to its snapshot")
		tops := [2]string{}
		for dir, l := range e.Ladders {
			tops[dir] = l[len(l)-1][0].Dec() + " -> " + l[len(l)-1][1].Dec()
		}
		t.Logf("%s, guarded, at block %d (unix %d), bid drift %d ppm/s, bid bucket %d a second: valid until %d; %d "+
			"checks in %d blocks, the state never above the door nor a size it refuses, at most %d and %d bp under it; "+
			"tops %v", p.Address, p.BlockNumber, p.Timestamp, drift, bucket, e.ValidUntil, n, blocks, worst[0], worst[1],
			tops)
	}
	require.NotZero(t, served, "no listing bound to a guarded entry of an approved source (EFFICIENT_PROP_SOURCES)")
	if out := os.Getenv("EFFICIENT_PROP_HOLD_OUT"); out != "" {
		b, err := json.MarshalIndent(checks, "", " ")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(out, b, 0o644))
	}
}
