package efficientprop

import (
	"bytes"
	"context"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
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
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/titan"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// noWordSelector is the front's NoWord(): a fill in a block whose desk word does not stand.
var noWordSelector = hexutil.Encode(crypto.Keccak256([]byte("NoWord()"))[:4])

// TestForkBuilderFillsTheSampledQuote runs on a local fork of a builder chain (Ethereum) that a stepper drives one
// block at a time, with a /state server answering for the stepper's next block. It lists the front's doors and, for
// each listing and direction in turn, refreshes it through /state and checks the simulator against the door's quote
// under the same overrides in the target block (at every sampled size, between them, outside them and at a sweep by the
// input token's decimals: never over the door, never a size the door refuses, equal at a sampled size); then lands the
// desks' updates, fills the ladder's first size from the taker in that block, and checks the fill paid the state's
// output to the wei. Then, per listing, with its desk's update left out of the next block: a refresh quotes nothing,
// and the same fill reverts NoWord there, moving none of the taker's tokens. A map whose entries hold the quote
// contracts alone, without the price contracts', quotes no listing that reads a print (on a fork whose listings read
// none, a WORD one, that case is logged as not applying). A map that the chain has passed (replayed after its block is
// mined) quotes nothing, nor does a map whose slot is a slot later than its block's, a carrying listing's word set to a
// 16 s life included, while the next poll of the true map quotes again. EFFICIENT_PROP_PRINTS, required, is how many of
// the fork's listings read a print (3 on the wstETH pool's PRINT dump, 0 on its WORD dump): exactly that many are
// quoted as reading one and checked on the map without the price contracts' entries, and every fill's class, which
// picks its gas, says the same. Where a listing carries (its binding's scopes bit 3), its word is set to carry with a
// 16 s life: the refresh samples the next block too, is valid to its second less the allowance, and a fill in that next
// block, with no update of that desk in it, pays the state's output; with an 11 s life the next block refuses every
// size, and the target block's ladder stands, valid to that block's second less the allowance.
//
// The fills go the way of the front's fill mode, EFFICIENT_PROP_FILL_MODE (default door): through the listing's door
// (swapFrom, the adapter's door entry), or through the front's own named entry (swapSender, which the adapter calls
// on a listing the front names no pool for, and which pays what its paid entry pays on any other). A listing with a
// credit mode is quoted only where EFFICIENT_PROP_CREDIT_LISTINGS names it (listing:custodyBaseIn:custodyQuoteIn:
// routeBaseIn:routeQuoteIn, comma separated): then each side's first size is filled, and its first routed size too
// where the side steps from custody to the route, each paying the state's output; a side its pool fills no size of
// (a route its pool's credit path leaves off) has no ladder, and the door refuses there. One not named is checked to
// have no ladder and left out of the rest. EFFICIENT_PROP_LISTING_GAS names listings' own gas (listing:baseIn:quoteIn).
//
//	EFFICIENT_PROP_RPC=<fork> EFFICIENT_PROP_FRONT=<front> EFFICIENT_PROP_FACTORY=<door factory> \
//	  EFFICIENT_PROP_CELL_CODEHASH=<the front's CELL_CODEHASH> EFFICIENT_PROP_SOURCES=<approved sources> \
//	  EFFICIENT_PROP_TITAN_URLS=<state urls> EFFICIENT_PROP_STEP=<stepper> EFFICIENT_PROP_TAKER=<taker> \
//	  EFFICIENT_PROP_PRINTS=<listings that read a print> \
//	  go test ./pkg/liquidity-source/efficient-prop -run TestForkBuilder -v
//
// The stepper is a command, split on spaces, run with one of: `set` with optional `--skip <desk contracts>`, `--carry
// <desk contracts>` (comma lists) and `--life <desk contract>=<s>` (the standing options of the blocks to come, which
// the /state server follows: whose updates are left out, whose words carry, a word's life; every option left out is
// reset), `send` (the next block's desk updates, sent and not mined) and `mine` (that block, on the chain's 12 s grid).
// The taker is an account the node signs for (an impersonated one on a fork), holding both tokens of every listing
// and approving every door. EFFICIENT_PROP_CHAIN sets the chain id (default Ethereum), EFFICIENT_PROP_MULTICALL the
// Multicall3 address and EFFICIENT_PROP_IMPLEMENTATIONS the implementations (default the one the proxy runs).
func TestForkBuilderFillsTheSampledQuote(t *testing.T) {
	rpcURL, front, factory, codehash := os.Getenv("EFFICIENT_PROP_RPC"), os.Getenv("EFFICIENT_PROP_FRONT"),
		os.Getenv("EFFICIENT_PROP_FACTORY"), os.Getenv("EFFICIENT_PROP_CELL_CODEHASH")
	urls, stepper, taker := os.Getenv("EFFICIENT_PROP_TITAN_URLS"), os.Getenv("EFFICIENT_PROP_STEP"),
		os.Getenv("EFFICIENT_PROP_TAKER")
	if rpcURL == "" || front == "" || factory == "" || codehash == "" || urls == "" || stepper == "" || taker == "" {
		t.Skip("the fork's variables, EFFICIENT_PROP_TITAN_URLS, EFFICIENT_PROP_STEP and EFFICIENT_PROP_TAKER are not " +
			"all set")
	}
	wantPrints, err := strconv.Atoi(os.Getenv("EFFICIENT_PROP_PRINTS"))
	require.NoError(t, err, "EFFICIENT_PROP_PRINTS, how many of the fork's listings read a print, is required")
	mc := multicall3
	if s := os.Getenv("EFFICIENT_PROP_MULTICALL"); s != "" {
		mc = common.HexToAddress(s)
	}
	ctx := context.Background()
	b := &builderFork{
		forkRun: forkRun{t: t, ctx: ctx, client: ethrpc.New(rpcURL).SetMulticallContract(mc)},
		urls:    strings.Split(urls, ","), stepper: strings.Fields(stepper), taker: common.HexToAddress(taker)}
	b.cfg = b.config(front, factory, codehash, b.urls)
	pools := b.list()

	// the target block's state: every listing quoted, the simulator never over the door, and a fill of its first size
	// in that block paid the state's output
	type filled struct {
		p    entity.Pool
		dir  int
		size *big.Int
	}
	var fills []filled
	// a listing with a credit mode the config does not name is not quoted, and is left out of the rest
	b.step("set")
	quoted := pools[:0:0]
	for _, p := range pools {
		q, _ := b.refresh(b.cfg, p)
		e := b.extra(q)
		if e.CreditMode != 0 && b.cfg.Fronts[0].creditListing(b.static(p).Listing) == nil {
			require.Empty(t, e.ladders()[0], "%s: a credit listing the config does not name is quoted", p.Address)
			require.Empty(t, e.ladders()[1], "%s: a credit listing the config does not name is quoted", p.Address)
			t.Logf("%s: listing %d has credit mode %d and no configured figures: not quoted", p.Address,
				b.static(p).Listing, e.CreditMode)
			continue
		}
		quoted = append(quoted, p)
	}
	pools = quoted
	require.NotEmpty(t, pools, "no listing the config quotes")
	require.LessOrEqual(t, wantPrints, len(pools), "EFFICIENT_PROP_PRINTS is over the fork's quoted listings")
	readsPrint := map[string]bool{}
	for k := range pools {
		for dir := range 2 {
			b.step("set")
			p, o := b.ready(b.cfg, pools[k], dir)
			e := b.extra(p)
			require.True(t, e.Builder)
			if e.Credit != nil && len(e.ladders()[dir]) == 0 {
				// a credit listing's side its pool fills no size of: the door refuses every size the grid holds
				b.checkUnder(p, o)
				t.Logf("%s direction %d: credit mode %d, no size filled on this side in block %d (custody %s, the "+
					"route off or refused): not quoted", p.Address, dir, e.CreditMode, o.number, p.Reserves[1-dir])
				continue
			}
			require.NotEmpty(t, e.ladders()[dir], "%s direction %d: no ladder in a block with its desk's update",
				p.Address, dir)
			require.Equal(t, o.time-b.cfg.builderAllowance(), e.ValidUntil,
				"%s: valid to the target block's second less the allowance", p.Address)
			b.checkUnder(p, o)
			size := e.ladders()[dir][0][0].ToBig()
			gas, routed := b.gasAt(p, dir, size)
			out := b.fillIn(p, dir, size, true, "")
			require.Equal(t, e.ladders()[dir][0][1].ToBig(), out, "%s direction %d: the fill pays the state's output",
				p.Address, dir)
			t.Logf("%s direction %d: filled %s -> %s in block %d, as sampled; declared gas %d (routed %v, print %v, "+
				"past %v, no asset %v, asset quote %v, own figures %v)", p.Address, dir, size, out, o.number, gas, routed,
				e.Print, e.Past, e.NoAsset, e.AssetQuote, e.Gas != nil)
			if c := e.Credit; c != nil && c.From[dir] > 0 && c.From[dir] < len(c.Ladders[dir]) {
				// the side steps from custody to the route: its first routed size too, from a new state
				b.step("set")
				p, o = b.ready(b.cfg, pools[k], dir)
				e = b.extra(p)
				if c := e.Credit; c != nil && c.From[dir] < len(c.Ladders[dir]) {
					pt := c.Ladders[dir][c.From[dir]]
					gas, routed := b.gasAt(p, dir, pt[0].ToBig())
					require.True(t, routed)
					out := b.fillIn(p, dir, pt[0].ToBig(), true, "")
					require.Equal(t, pt[1].ToBig(), out, "%s direction %d: the routed fill pays the state's output",
						p.Address, dir)
					t.Logf("%s direction %d: filled %s -> %s in block %d through the credit route, as sampled; "+
						"declared gas %d", p.Address, dir, pt[0], out, o.number, gas)
				}
			}
			switch wantPrints {
			case 0:
				require.False(t, e.Print, "%s direction %d: charged as reading a print on a fork with none", p.Address,
					dir)
			case len(pools):
				require.True(t, e.Print, "%s direction %d: charged as reading no print on a fork whose listings all "+
					"read one", p.Address, dir)
			}
			if was, seen := readsPrint[p.Address]; seen {
				require.Equal(t, was, e.Print, "%s: one class on both sides", p.Address)
			}
			readsPrint[p.Address] = e.Print
			fills = append(fills, filled{p: p, dir: dir, size: size})
		}
	}

	// a block without a listing's desk update: the listing is not quoted, and the fill reverts NoWord
	for _, f := range fills {
		if f.dir != 0 {
			continue
		}
		cell := b.cellOf(f.p)
		b.step("set", "--skip", cell.Hex())
		p, o := b.refresh(b.cfg, f.p)
		e := b.extra(p)
		require.Empty(t, e.ladders()[0], "%s: quoted with no update of its desk in the target block", p.Address)
		require.Empty(t, e.ladders()[1], "%s: quoted with no update of its desk in the target block", p.Address)
		for _, other := range pools {
			if other.Address != p.Address && b.cellOf(other) != cell {
				q, _ := b.ready(b.cfg, other, -1)
				eq := b.extra(q)
				require.NotZero(t, len(eq.ladders()[0])+len(eq.ladders()[1]),
					"%s: another desk's missing update unquotes it", other.Address)
				break
			}
		}
		_, quoted := b.quoteUnder(p.Address, p.Tokens[0].Address, p.Tokens[1].Address, f.size, p.BlockNumber, o)
		require.False(t, quoted, "%s: the door quotes in a block without its desk's update", p.Address)
		b.fillIn(p, 0, f.size, false, noWordSelector)
		t.Logf("%s: no update of desk contract %s in block %d: no ladder, the fill of %s reverted NoWord",
			p.Address, cell.Hex(), o.number, f.size)
	}

	// a map whose entries hold the quote contracts alone: no listing that reads a print is quoted, though each quote
	// contract has its entry and the whole map quotes it. The head holds every pair's update, so a print whose window
	// outlives a block (the stable pair's) still prices the door at the head's print under that map
	b.step("set")
	cells := map[common.Address]bool{}
	for _, p := range pools {
		cells[b.cellOf(p)] = true
	}
	partial := b.partial(func(a common.Address) bool { return cells[a] })
	noPrint := b.config(front, factory, codehash, []string{partial.URL})
	prints := 0
	for _, p := range pools {
		if !printed(b.view(p).Scopes) {
			continue
		}
		b.step("send")
		b.step("mine")
		q, _ := b.ready(b.cfg, p, -1)
		eq := b.extra(q)
		require.NotZero(t, len(eq.ladders()[0])+len(eq.ladders()[1]), "%s: not quoted on the whole map", p.Address)
		q, o := b.refresh(noPrint, p)
		e := b.extra(q)
		require.Empty(t, e.ladders()[0], "%s: quoted with no entry of its price contract", p.Address)
		require.Empty(t, e.ladders()[1], "%s: quoted with no entry of its price contract", p.Address)
		dir := 0
		if len(eq.ladders()[0]) == 0 {
			dir = 1
		}
		size := eq.ladders()[dir][0][0].ToBig()
		whole, ok := b.quoteUnder(p.Address, p.Tokens[dir].Address, p.Tokens[1-dir].Address, size, q.BlockNumber,
			b.state(b.cfg))
		require.True(t, ok, "%s: the door refuses under the whole map", p.Address)
		without, ok := b.quoteUnder(p.Address, p.Tokens[dir].Address, p.Tokens[1-dir].Address, size, q.BlockNumber, o)
		t.Logf("%s: a map of block %d without its price contract's entry: no ladder; the door at %s in there: %v %s, "+
			"under the whole map %s", p.Address, o.number, size, ok, without, whole)
		prints++
	}
	t.Logf("%d listings read a print", prints)
	require.Equal(t, wantPrints, prints, "listings read as reading a print")
	charged := 0
	for _, reads := range readsPrint {
		if reads {
			charged++
		}
	}
	require.Equal(t, wantPrints, charged, "listings whose fills were charged as reading a print")

	// a map the chain has passed: replayed after its block is mined, it is not ahead of the head
	b.step("set")
	replay := b.replay()
	stale := b.config(front, factory, codehash, []string{replay.URL})
	b.ready(b.cfg, pools[0], -1)
	p, _ := b.refresh(stale, pools[0])
	e := b.extra(p)
	require.NotZero(t, len(e.ladders()[0])+len(e.ladders()[1]),
		"the replayed map quotes while its block is ahead of the head")
	b.step("send")
	b.step("mine")
	for _, p := range pools {
		p, _ = b.refresh(stale, p)
		e := b.extra(p)
		require.Empty(t, e.ladders()[0], "%s: quoted on a map the chain has passed", p.Address)
		require.Empty(t, e.ladders()[1], "%s: quoted on a map the chain has passed", p.Address)
		require.Zero(t, e.ValidUntil)
	}
	t.Logf("a map of block %d replayed at head %d: no listing quoted", replay.number, b.head().Number)

	// a map whose slot is a slot later than its block's (a missed slot, or an answer ahead of the wall clock): no
	// listing is quoted, a carrying one with a 16 s life included, whose door still fills at that later second
	b.laterSlot(pools)

	// carry: a listing whose binding carries, its word set to carry with a 16 s life
	for _, p := range pools {
		if v := b.view(p); (v.Scopes>>3)&1 == 1 && (v.Scopes>>1)&3 != fenceNone {
			b.carry(p)
		}
	}
	b.step("set")
}

// builderFork is the fork test's chain, its /state urls, its stepper and its taker.
type builderFork struct {
	forkRun
	cfg     *Config
	urls    []string
	stepper []string
	taker   common.Address
}

// config is the fork's front, read through /state at urls and filled in EFFICIENT_PROP_FILL_MODE (default the
// door's), with the credit listings and the listings' own gas the environment names.
func (b *builderFork) config(front, factory, codehash string, urls []string) *Config {
	f := forkFront(front, factory)
	f.FillMode, f.BaseTime, f.Layout, f.CellCodehash = FillModeDoor, "", LayoutBuilder, codehash
	if m := os.Getenv("EFFICIENT_PROP_FILL_MODE"); m != "" {
		f.FillMode = m
	}
	for _, s := range strings.FieldsFunc(os.Getenv("EFFICIENT_PROP_CREDIT_LISTINGS"), isComma) {
		v := b.ints(s, 5)
		f.CreditListings = append(f.CreditListings, CreditListing{Listing: uint32(v[0]), CustodyBaseIn: v[1],
			CustodyQuoteIn: v[2], RouteBaseIn: v[3], RouteQuoteIn: v[4]})
	}
	for _, s := range strings.FieldsFunc(os.Getenv("EFFICIENT_PROP_LISTING_GAS"), isComma) {
		v := b.ints(s, 3)
		f.ListingGas = append(f.ListingGas, ListingGas{Listing: uint32(v[0]), BaseIn: v[1], QuoteIn: v[2]})
	}
	f.Titan = &titan.Config{URLs: urls, Timeout: 10 * time.Second}
	chain := valueobject.ChainIDEthereum
	if s := os.Getenv("EFFICIENT_PROP_CHAIN"); s != "" {
		id, _ := strconv.ParseUint(s, 10, 64)
		chain = valueobject.ChainID(id)
	}
	cfg := &Config{ChainID: chain, Fronts: []Front{f}}
	require.NoError(b.t, cfg.validate())
	return cfg
}

// list is the front's listed doors, each token's decimals set.
func (b *builderFork) list() []entity.Pool {
	t := b.t
	u, err := NewPoolsListUpdater(b.cfg, b.client)
	require.NoError(t, err)
	pools, _, err := u.GetNewPools(b.ctx, nil)
	require.NoError(t, err)
	require.NotEmpty(t, pools, "the front has no opened listing door")
	for _, p := range pools {
		for _, tok := range p.Tokens {
			var dec uint8
			_, err := b.client.NewRequest().SetContext(b.ctx).
				AddCall(&ethrpc.Call{ABI: abi.Erc20ABI, Target: tok.Address, Method: "decimals"}, []any{&dec}).Call()
			require.NoError(t, err)
			tok.Decimals = dec
		}
	}
	return pools
}

// step runs the stepper with args.
func (b *builderFork) step(args ...string) {
	cmd := exec.CommandContext(b.ctx, b.stepper[0], append(b.stepper[1:], args...)...)
	out, err := cmd.CombinedOutput()
	require.NoError(b.t, err, "stepper %v: %s", args, out)
}

func isComma(r rune) bool { return r == ',' }

// ints is s, n integers separated by colons.
func (b *builderFork) ints(s string, n int) []int64 {
	parts := strings.Split(s, ":")
	require.Len(b.t, parts, n, "%q: want %d integers separated by colons", s, n)
	out := make([]int64, n)
	for i, x := range parts {
		v, err := strconv.ParseInt(x, 10, 64)
		require.NoError(b.t, err, s)
		out[i] = v
	}
	return out
}

// static is p's static extra.
func (b *builderFork) static(p entity.Pool) StaticExtra {
	var se StaticExtra
	require.NoError(b.t, json.Unmarshal([]byte(p.StaticExtra), &se))
	return se
}

// state is the /state answer for the next block, as the tracker reads it (fetchAnswers: the first url's answer for
// the highest block), as a call override. An answer that holds no entry, as the api gives where no desk's update is in
// the block, is an answer: its block and second stand, with no account.
func (b *builderFork) state(cfg *Config) override {
	var clients []stateClient
	for _, url := range cfg.Fronts[0].Titan.URLs {
		c, err := rpc.DialContext(b.ctx, url)
		require.NoError(b.t, err)
		b.t.Cleanup(c.Close)
		clients = append(clients, stateClient{url: url, client: c})
	}
	answers := fetchAnswers(b.ctx, clients, cfg.Fronts[0].Titan.Timeout, nil)
	var a *answer
	for k := range answers {
		if answers[k].number != nil && (a == nil || answers[k].number.Cmp(a.number) > 0) {
			a = &answers[k]
		}
	}
	require.NotNil(b.t, a, "no /state answer")
	require.NotZero(b.t, a.time, "the /state answer names no slot")
	return override{time: a.time, number: a.number.Uint64(),
		state: &titan.State{Overrides: a.accounts, BlockNumber: a.number, BlockTimestamp: a.time}}
}

// refresh refreshes p under cfg twice, the second sizing from the first's ladder, and answers the /state answer it
// was read on. A /state server that traces the next block on the fork itself moves the fork's head while it traces,
// so the answer is read once first, and the refreshes read it again from the server's cache.
func (b *builderFork) refresh(cfg *Config, p entity.Pool) (entity.Pool, override) {
	t := b.t
	o := b.state(cfg)
	tr, err := NewPoolTracker(cfg, b.client)
	require.NoError(t, err)
	for range 2 {
		p, err = tr.GetNewPoolState(b.ctx, p, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
	}
	return p, o
}

// readyBlocks bounds how many blocks ready waits for a side to refill.
const readyBlocks = 40

// ready refreshes p until direction dir (either, where dir is -1) has a ladder, landing a block of the desks' updates
// with no fill between two tries: a fill empties its side's rate ceiling, which refills over time.
func (b *builderFork) ready(cfg *Config, p entity.Pool, dir int) (entity.Pool, override) {
	for k := 0; ; k++ {
		q, o := b.refresh(cfg, p)
		e := b.extra(q)
		if dir < 0 && len(e.ladders()[0])+len(e.ladders()[1]) > 0 || dir >= 0 && len(e.ladders()[dir]) > 0 ||
			k == readyBlocks {
			if k > 0 {
				b.t.Logf("%s: waited %d blocks for a ladder", p.Address, k)
			}
			return q, o
		}
		b.step("send")
		b.step("mine")
	}
}

// quoteUnder is the door's quote pinned to block, run in o's block on its state; ok false where the door refuses.
func (b *builderFork) quoteUnder(door, tokenIn, tokenOut string, size *big.Int, block uint64,
	o override) (*big.Int, bool) {
	data, err := doorABI.Pack("quote", common.HexToAddress(tokenIn), common.HexToAddress(tokenOut), size)
	require.NoError(b.t, err)
	var raw hexutil.Bytes
	args := []any{map[string]any{"to": common.HexToAddress(door), "data": hexutil.Bytes(data)},
		hexutil.EncodeUint64(block)}
	if err := b.client.GetETHClient().Client().CallContext(b.ctx, &raw, "eth_call",
		append(args, overrideArgs(o)...)...); err != nil {
		return nil, false
	}
	vals, err := doorABI.Unpack("quote", raw)
	require.NoError(b.t, err)
	return vals[0].(*big.Int), true
}

// checkUnder prices p's simulator against the door under o, in its target block, both ways: never over the door,
// never a size the door refuses, and the door's answer at each sampled size.
func (b *builderFork) checkUnder(p entity.Pool, o override) {
	t := b.t
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
	require.NoError(t, err)
	e := b.extra(p)
	for dir := range 2 {
		tokenIn, tokenOut := p.Tokens[dir].Address, p.Tokens[1-dir].Address
		n := 0
		for _, size := range sizes(p, e, dir) {
			res, simErr := calc(sim, tokenIn, tokenOut, size, nil)
			quote, quoted := b.quoteUnder(p.Address, tokenIn, tokenOut, size, p.BlockNumber, o)
			if simErr != nil {
				continue
			}
			n++
			require.True(t, quoted, "%s: the simulator quotes %s in where the door refuses", p.Address, size)
			require.LessOrEqual(t, res.TokenAmountOut.Amount.Cmp(quote), 0,
				"%s: at %s in the simulator pays %s, the door %s", p.Address, size, res.TokenAmountOut.Amount, quote)
		}
		for _, pt := range e.ladders()[dir] {
			quote, quoted := b.quoteUnder(p.Address, tokenIn, tokenOut, pt[0].ToBig(), p.BlockNumber, o)
			require.True(t, quoted)
			require.Equal(t, quote, pt[1].ToBig(), "%s: a sampled size's output is the door's in the target block",
				p.Address)
		}
		b.t.Logf("%s direction %d: %d sizes quoted, none over the door in block %d", p.Address, dir, n, o.number)
	}
}

// fillIn lands the next block: the desks' updates, then a fill of size from the taker, then the block. In the door's
// mode the fill goes through p's door (swapFrom, the adapter's door entry); in the front's, through the front's named
// entry (swapSender with the listing, the taker's grant the front's). It answers what the taker received of the output
// token; where ok is false the fill must revert with the error selector want, the taker's tokens unmoved.
func (b *builderFork) fillIn(p entity.Pool, dir int, size *big.Int, ok bool, want string) *big.Int {
	t := b.t
	tokenIn, tokenOut := common.HexToAddress(p.Tokens[dir].Address), common.HexToAddress(p.Tokens[1-dir].Address)
	head := b.head()
	word := func(x []byte) []byte { return common.LeftPadBytes(x, 32) }
	to := common.HexToAddress(p.Address)
	data := crypto.Keccak256([]byte("swapFrom(address,address,uint256,uint256,address,uint256)"))[:4]
	args := [][]byte{tokenIn[:], tokenOut[:], size.Bytes(), {1}, b.taker[:],
		new(big.Int).SetUint64(head.Time + 120).Bytes()}
	if se := b.static(p); se.Mode == modeFront {
		to = common.HexToAddress(se.Front)
		data = crypto.Keccak256([]byte("swapSender(address,address,uint256,uint256,address,uint256,uint24)"))[:4]
		args = append(args, new(big.Int).SetUint64(uint64(se.Listing)).Bytes())
	}
	for _, x := range args {
		data = append(data, word(x)...)
	}
	b.step("send")
	var hash common.Hash
	require.NoError(t, b.client.GetETHClient().Client().CallContext(b.ctx, &hash, "eth_sendTransaction",
		map[string]any{"from": b.taker, "to": to, "data": hexutil.Bytes(data),
			"gas": hexutil.Uint64(2_000_000)}))
	b.step("mine")
	r, err := b.client.GetETHClient().TransactionReceipt(b.ctx, hash)
	require.NoError(t, err)
	require.Equal(t, head.Number.Uint64()+1, r.BlockNumber.Uint64(), "the fill landed in the stepper's block")
	balances := func(n *big.Int) [2]*big.Int {
		var out [2]*big.Int
		for k, tok := range []common.Address{tokenIn, tokenOut} {
			out[k] = new(big.Int)
			_, err := b.client.NewRequest().SetContext(b.ctx).SetBlockNumber(n).
				AddCall(&ethrpc.Call{ABI: abi.Erc20ABI, Target: tok.Hex(), Method: "balanceOf",
					Params: []any{b.taker}}, []any{&out[k]}).Call()
			require.NoError(t, err)
		}
		return out
	}
	before, after := balances(head.Number), balances(r.BlockNumber)
	if !ok {
		require.Equal(t, types.ReceiptStatusFailed, r.Status, "%s: the fill landed with no update of its desk",
			p.Address)
		require.Equal(t, before, after, "%s: a reverted fill moved the taker's tokens", p.Address)
		b.requireRevert(hash, r.BlockNumber, want)
		return nil
	}
	require.Equal(t, types.ReceiptStatusSuccessful, r.Status, "%s: the fill reverted in its desk's block", p.Address)
	require.Equal(t, size, new(big.Int).Sub(before[0], after[0]), "%s: the fill took its input", p.Address)
	return new(big.Int).Sub(after[1], before[1])
}

// requireRevert replays a mined transaction at its block's state before it and checks its revert selector.
func (b *builderFork) requireRevert(hash common.Hash, block *big.Int, want string) {
	t := b.t
	tx, _, err := b.client.GetETHClient().TransactionByHash(b.ctx, hash)
	require.NoError(t, err)
	var raw hexutil.Bytes
	err = b.client.GetETHClient().Client().CallContext(b.ctx, &raw, "eth_call", map[string]any{
		"from": b.taker, "to": tx.To(), "data": hexutil.Bytes(tx.Data()), "gas": hexutil.Uint64(tx.Gas())},
		hexutil.EncodeBig(block))
	require.Error(t, err, "the replayed fill succeeds")
	var de rpc.DataError
	require.ErrorAs(t, err, &de)
	data, _ := de.ErrorData().(string)
	require.True(t, strings.HasPrefix(data, want), "revert data %s, want %s", data, want)
}

func (b *builderFork) head() *types.Header {
	h, err := b.client.GetETHClient().HeaderByNumber(b.ctx, nil)
	require.NoError(b.t, err)
	return h
}

// view is the listing's doorView at the head.
func (b *builderFork) view(p entity.Pool) DoorView {
	var se StaticExtra
	require.NoError(b.t, json.Unmarshal([]byte(p.StaticExtra), &se))
	var out doorViewOut
	_, err := b.client.NewRequest().SetContext(b.ctx).AddCall(&ethrpc.Call{ABI: frontABI, Target: se.Front,
		Method: "doorView", Params: []any{big.NewInt(int64(se.Listing))}}, []any{&out}).Call()
	require.NoError(b.t, err)
	return out.View
}

// cellOf is the listing's desk contract (the front's deskCellOf of its doorView's desk).
func (b *builderFork) cellOf(p entity.Pool) common.Address {
	var se StaticExtra
	require.NoError(b.t, json.Unmarshal([]byte(p.StaticExtra), &se))
	var cell common.Address
	_, err := b.client.NewRequest().SetContext(b.ctx).AddCall(&ethrpc.Call{ABI: frontABI, Target: se.Front,
		Method: "deskCellOf", Params: []any{b.view(p).Desk}}, []any{&cell}).Call()
	require.NoError(b.t, err)
	require.NotEqual(b.t, common.Address{}, cell)
	return cell
}

// gasAt is the gas the simulator declares for a fill of size in p's direction dir, and whether it is a credit route.
func (b *builderFork) gasAt(p entity.Pool, dir int, size *big.Int) (int64, bool) {
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
	require.NoError(b.t, err)
	res, err := calc(sim, p.Tokens[dir].Address, p.Tokens[1-dir].Address, size, nil)
	require.NoError(b.t, err)
	info, _ := res.SwapInfo.(SwapInfo)
	return res.Gas, info.Routed
}

// carry checks a carrying listing: with its word carried and its life 16 s, the refresh samples the target block and
// the next under the same slots, keeps the lower output, and is valid to the next block's second less the allowance;
// a fill in that next block, which holds no update of the desk, pays the state's output.
func (b *builderFork) carry(p entity.Pool) {
	t := b.t
	cell := b.cellOf(p).Hex()
	b.step("set", "--carry", cell, "--life", cell+"=16")
	p, o := b.ready(b.cfg, p, -1)
	e := b.extra(p)
	dir := 0
	if len(e.ladders()[0]) == 0 {
		dir = 1
	}
	require.NotEmpty(t, e.ladders()[dir], "%s: a carried word quotes nothing", p.Address)
	require.Equal(t, o.time+slotSeconds-b.cfg.builderAllowance(), e.ValidUntil,
		"%s: valid to the carried block's second less the allowance", p.Address)
	next := o
	next.time, next.number = o.time+slotSeconds, o.number+1
	for dir := range 2 {
		tokenIn, tokenOut := p.Tokens[dir].Address, p.Tokens[1-dir].Address
		for _, pt := range e.ladders()[dir] {
			at, ok := b.quoteUnder(p.Address, tokenIn, tokenOut, pt[0].ToBig(), p.BlockNumber, o)
			require.True(t, ok)
			then, ok := b.quoteUnder(p.Address, tokenIn, tokenOut, pt[0].ToBig(), p.BlockNumber, next)
			require.True(t, ok, "%s: a size the carried block refuses", p.Address)
			low := at
			if then.Cmp(low) < 0 {
				low = then
			}
			require.Equal(t, low, pt[1].ToBig(), "%s: the lower of the two blocks' outputs", p.Address)
		}
	}
	b.step("send")
	b.step("mine")
	b.step("set", "--skip", cell, "--carry", cell, "--life", cell+"=16")
	size := e.ladders()[dir][0][0].ToBig()
	out := b.fillIn(p, dir, size, true, "")
	require.Equal(t, e.ladders()[dir][0][1].ToBig(), out, "%s: the fill in the carried block pays the state's output",
		p.Address)
	t.Logf("%s direction %d: carried to block %d (valid until %d): filled %s -> %s there, with no update of its desk",
		p.Address, dir, next.number, e.ValidUntil, size, out)

	// with an 11 s life the word is past its life in the carried block, which refuses every size: the target block's
	// ladder stands, valid to its second less the allowance
	b.step("set", "--carry", cell)
	q, o := b.ready(b.cfg, p, -1)
	eq := b.extra(q)
	dir = 0
	if len(eq.ladders()[0]) == 0 {
		dir = 1
	}
	require.NotEmpty(t, eq.ladders()[dir], "%s: an 11 s carried word quotes nothing in its own block", q.Address)
	require.Equal(t, o.time-b.cfg.builderAllowance(), eq.ValidUntil, "%s: valid to the target block alone", q.Address)
	tokenIn, tokenOut := q.Tokens[dir].Address, q.Tokens[1-dir].Address
	for _, pt := range eq.ladders()[dir] {
		at, ok := b.quoteUnder(q.Address, tokenIn, tokenOut, pt[0].ToBig(), q.BlockNumber, o)
		require.True(t, ok)
		require.Equal(t, at, pt[1].ToBig(), "%s: the target block's output", q.Address)
		_, ok = b.quoteUnder(q.Address, tokenIn, tokenOut, pt[0].ToBig(), q.BlockNumber,
			override{time: o.time + slotSeconds, number: o.number + 1, state: o.state})
		require.False(t, ok, "%s: the carried block fills a word past its life", q.Address)
	}
	t.Logf("%s: carried word with an 11 s life: the carried block refuses; %d points of block %d, valid until %d",
		q.Address, len(eq.ladders()[dir]), o.number, eq.ValidUntil)
}

// partial is a /state server answering the fork's /state server's answers with only the accounts keep allows in each
// entry.
func (b *builderFork) partial(keep func(common.Address) bool) *httptest.Server {
	return b.proxy(func(result map[string]json.RawMessage) {
		for k, v := range result {
			if k == "slot" || k == "blockNumber" {
				continue
			}
			var e struct {
				StateOverride map[string]json.RawMessage `json:"stateOverride"`
			}
			if json.Unmarshal(v, &e) != nil {
				continue
			}
			for a := range e.StateOverride {
				if !keep(common.HexToAddress(a)) {
					delete(e.StateOverride, a)
				}
			}
			result[k], _ = json.Marshal(e)
		}
	})
}

// shifted is a /state server answering the fork's /state server's answers with their slot moved by later slots.
func (b *builderFork) shifted(later uint64) *httptest.Server {
	return b.proxy(func(result map[string]json.RawMessage) {
		var slot uint64
		if json.Unmarshal(result["slot"], &slot) == nil {
			result["slot"], _ = json.Marshal(slot + later)
		}
	})
}

// proxy is a /state server answering the fork's /state server's answers, each result edited by edit.
func (b *builderFork) proxy(edit func(result map[string]json.RawMessage)) *httptest.Server {
	t := b.t
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		hr, err := http.NewRequestWithContext(b.ctx, http.MethodPost, b.urls[0], bytes.NewReader(body))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		hr.Header.Set("content-type", "application/json")
		resp, err := http.DefaultClient.Do(hr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer func() { _ = resp.Body.Close() }()
		var answer struct {
			ID     json.RawMessage            `json:"id"`
			Result map[string]json.RawMessage `json:"result"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&answer); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		edit(answer.Result)
		out, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": answer.ID, "result": answer.Result})
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write(out)
	}))
	t.Cleanup(s.Close)
	return s
}

// laterSlot checks that a map whose slot is a slot later than its block's quotes nothing: every listing, with each
// carrying listing's word set to carry with a 16 s life, is quoted on the true map, then not on the shifted one. For a
// carrying listing it logs the door's answer at the later second, at the true ladder's top: a size the door may fill
// there but not in the true target block. The next poll of the true map quotes again.
func (b *builderFork) laterSlot(pools []entity.Pool) {
	t := b.t
	args := []string{"set"}
	var carrying, lives []string
	for _, p := range pools {
		if v := b.view(p); (v.Scopes>>3)&1 == 1 && (v.Scopes>>1)&3 != fenceNone {
			cell := b.cellOf(p).Hex()
			carrying, lives = append(carrying, cell), append(lives, cell+"=16")
		}
	}
	if len(carrying) > 0 {
		args = append(args, "--carry", strings.Join(carrying, ","), "--life", strings.Join(lives, ","))
	}
	b.step(args...)
	later := b.shifted(1)
	shifted := b.config(b.cfg.Fronts[0].Address, b.cfg.Fronts[0].DoorFactory, b.cfg.Fronts[0].CellCodehash,
		[]string{later.URL})
	for _, p := range pools {
		q, o := b.ready(b.cfg, p, -1)
		eq := b.extra(q)
		require.NotZero(t, len(eq.ladders()[0])+len(eq.ladders()[1]), "%s: not quoted on the true map", p.Address)
		s, _ := b.refresh(shifted, q)
		e := b.extra(s)
		require.Empty(t, e.ladders()[0], "%s: quoted on a map a slot later than its block's", p.Address)
		require.Empty(t, e.ladders()[1], "%s: quoted on a map a slot later than its block's", p.Address)
		require.Zero(t, e.ValidUntil, "%s: a later slot's state is valid", p.Address)
		dir := 0
		if len(eq.ladders()[0]) == 0 {
			dir = 1
		}
		top := eq.ladders()[dir][len(eq.ladders()[dir])-1][0].ToBig()
		at := o
		at.time += slotSeconds
		out, ok := b.quoteUnder(p.Address, p.Tokens[dir].Address, p.Tokens[1-dir].Address, top, q.BlockNumber, at)
		t.Logf("%s: a map of block %d a slot later: no ladder (true map: %d+%d points, valid until %d); the door at "+
			"the later second at the true top %s: %v %s", p.Address, o.number, len(eq.ladders()[0]), len(eq.ladders()[1]),
			eq.ValidUntil, top, ok, out)
	}
	b.step("set")
	q, _ := b.ready(b.cfg, pools[0], -1)
	eq := b.extra(q)
	require.NotZero(t, len(eq.ladders()[0])+len(eq.ladders()[1]), "the next poll of the true map quotes nothing")
}

// replayServer answers titan_getPammStateOverrides with one answer, read from the /state server once.
type replayServer struct {
	*httptest.Server
	number uint64
}

// replay is a server that answers the /state server's current answer from now on.
func (b *builderFork) replay() *replayServer {
	t := b.t
	req := []byte(`{"jsonrpc":"2.0","id":1,"method":"titan_getPammStateOverrides","params":[]}`)
	hr, err := http.NewRequestWithContext(b.ctx, http.MethodPost, b.urls[0], bytes.NewReader(req))
	require.NoError(t, err)
	hr.Header.Set("content-type", "application/json")
	resp, err := http.DefaultClient.Do(hr)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	var answer struct {
		Result map[string]json.RawMessage `json:"result"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&answer))
	result, err := json.Marshal(answer.Result)
	require.NoError(t, err)
	var number hexutil.Uint64
	require.NoError(t, json.Unmarshal(answer.Result["blockNumber"], &number))
	s := &replayServer{number: uint64(number)}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var in struct {
			ID json.RawMessage `json:"id"`
		}
		_ = json.Unmarshal(body, &in)
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(in.ID) + `,"result":` + string(result) + `}`))
	}))
	t.Cleanup(s.Close)
	return s
}

// TestForkBuilderMapsAFillsSwappedToItsDoor runs against an Ethereum fork (or the chain) with a front of the builder
// build and a block in which a route filled a listing through its door, EFFICIENT_PROP_FILL_BLOCK. It hands that
// block's logs to the event parser and checks the fill's Swapped as the deployed contracts log it: the listing's quote
// contract logs it and no door does, it names the door its sender, and the parser maps it to that door alone. It
// needs no /state read and sends nothing.
//
//	EFFICIENT_PROP_RPC=<fork> EFFICIENT_PROP_FRONT=<front> EFFICIENT_PROP_FACTORY=<door factory> \
//	  EFFICIENT_PROP_CELL_CODEHASH=<the front's CELL_CODEHASH> EFFICIENT_PROP_FILL_BLOCK=<block> \
//	  go test ./pkg/liquidity-source/efficient-prop -run TestForkBuilderMapsAFillsSwapped -v
func TestForkBuilderMapsAFillsSwappedToItsDoor(t *testing.T) {
	rpcURL, front, factory, codehash := os.Getenv("EFFICIENT_PROP_RPC"), os.Getenv("EFFICIENT_PROP_FRONT"),
		os.Getenv("EFFICIENT_PROP_FACTORY"), os.Getenv("EFFICIENT_PROP_CELL_CODEHASH")
	at, err := strconv.ParseUint(os.Getenv("EFFICIENT_PROP_FILL_BLOCK"), 10, 64)
	if rpcURL == "" || front == "" || factory == "" || codehash == "" || err != nil {
		t.Skip("the fork's variables and EFFICIENT_PROP_FILL_BLOCK, a block holding a fill through a door, are not " +
			"all set")
	}
	ctx := context.Background()
	b := &builderFork{forkRun: forkRun{t: t, ctx: ctx, client: ethrpc.New(rpcURL).SetMulticallContract(multicall3)}}
	// the parser dials no /state url: the config's is never read here
	cfg := b.config(front, factory, codehash, []string{"http://127.0.0.1:1"})
	block := new(big.Int).SetUint64(at)
	logs, err := b.client.GetETHClient().FilterLogs(ctx, ethereum.FilterQuery{FromBlock: block, ToBlock: block})
	require.NoError(t, err)
	p := NewPoolFactory(cfg, b.client)
	mapped, err := p.Decode(ctx, logs)
	require.NoError(t, err)
	doors := p.knownDoors(ctx, func(l learned) []string { return l.all[common.HexToAddress(front)] })
	require.NotEmpty(t, doors, "the front has reserved no listing")
	var fills int
	for _, l := range logs {
		if len(l.Topics) != 4 || l.Topics[0] != topicSwapped {
			continue
		}
		emitter, sender := hexutil.Encode(l.Address[:]), hexutil.Encode(l.Topics[1][12:])
		require.NotContains(t, doors, emitter, "a door logged Swapped on a builder build")
		if !slices.Contains(doors, sender) {
			continue // another venue's fill, or one through the front's own entries
		}
		fills++
		for door, mine := range mapped {
			has := slices.ContainsFunc(mine, func(m types.Log) bool {
				return m.Index == l.Index && m.TxHash == l.TxHash && m.Topics[0] == topicSwapped
			})
			require.Equal(t, door == sender, has, "the Swapped of %s (sender %s) and door %s", emitter, sender, door)
		}
		require.Contains(t, mapped, sender, "the fill's door is refreshed")
		t.Logf("block %d: Swapped logged by %s, sender (the door) %s, mapped to that door alone", at, emitter, sender)
	}
	require.NotZero(t, fills, "block %d holds no fill through a door of the front", at)
}
