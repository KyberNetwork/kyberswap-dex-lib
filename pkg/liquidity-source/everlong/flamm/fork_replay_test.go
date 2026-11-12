package everlongflamm

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"maps"
	"math/big"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// forkFixture is a refresh recorded on a local fork of its chain with pools deployed by the protocol's own deploy
// scripts: every call the lister and the tracker sent (a multicall's calls each) with what the chain returned, the hooks' code,
// the pools as the tracker left them, and fills of the pool's own swap through the KyberSwap adapter at the
// refresh block, at every probed size and between them.
type forkFixture struct {
	ChainID uint64              `json:"chainId"`
	Config  Config              `json:"config"`
	Kinds   map[string]string   `json:"kinds"` // pool: the pool's kind (its hook and lending venue)
	Calls   map[string]recorded `json:"calls"` // "from to data", lower case
	Code    map[string]string   `json:"code"`
	Storage map[string]string   `json:"storage,omitempty"` // "address slot": the word read
	Pools   []entity.Pool       `json:"pools"`             // the first refresh
	Warm    []entity.Pool       `json:"warm"`              // the second, sized from the first's tops
	Swaps   []forkSwap          `json:"swaps"`
	// Volatile is a refresh of each pool with its hook's variance fold set past the fee law's cap, Pairs two swaps in
	// a row on either state.
	Volatile []entity.Pool `json:"volatile"`
	Pairs    []forkPair    `json:"pairs"`
	// Capped is a refresh of each pool with token 1's notional cap lowered, so that it cuts a sale's output and stops
	// a buy's input, CappedSwaps full-amount fills of that state through the adapter.
	Capped      []entity.Pool `json:"capped"`
	CappedSwaps []forkSwap    `json:"cappedSwaps"`
	// Sweeps are fills of each direction at sweepSizes of its ladders, Edges fills of each pool's sale across the
	// far edge of its leverage route (fork_lever_test.go).
	Sweeps []forkSweep `json:"sweeps,omitempty"`
	Edges  []forkEdge  `json:"edges,omitempty"`
}

type recorded struct {
	OK  bool   `json:"ok"`
	Ret string `json:"ret,omitempty"`
}

// forkSwap is one fill of the pool's swap through the adapter: the input the pool did not take and the output.
type forkSwap struct {
	Pool         string `json:"pool"`
	TokenIn      string `json:"tokenIn"`
	TokenOut     string `json:"tokenOut"`
	AmountIn     string `json:"amountIn"`
	AmountUnused string `json:"amountUnused"`
	AmountOut    string `json:"amountOut"`
	Rung         bool   `json:"rung,omitempty"`    // a size the refresh probed
	Reverts      bool   `json:"reverts,omitempty"` // the swap reverted: no amounts
	Refresh      int    `json:"refresh,omitempty"` // the refresh whose ladders it is checked against
}

// forkPair is two swaps of one pool through the adapter in one block, the second on the state the first left: the
// chain's own answer to a route that crosses the pool twice.
type forkPair struct {
	Volatile bool     `json:"volatile,omitempty"` // on the state of forkFixture.Volatile, else the first refresh's
	First    forkSwap `json:"first"`
	Second   forkSwap `json:"second"`
}

func callKey(from, to common.Address, data []byte) string {
	return strings.ToLower(from.Hex() + " " + to.Hex() + " " + hexutil.Encode(data))
}

func loadFixture(t *testing.T, path string) *forkFixture {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	z, err := gzip.NewReader(f)
	require.NoError(t, err)
	var fx forkFixture
	require.NoError(t, json.NewDecoder(z).Decode(&fx))
	return &fx
}

// replayNode answers from the fixture's calls as a chain at block, the sampler's call as recorded among them; a call
// it lacks fails the test, but three reads of a fixture recorded before the tracker made them, which it answers as
// that fixture's venues would: the Router's venueActions reverts, as on a Router without it, which leaves no repay
// certain; a venue account's capabilities() answers none, as its Morpho Blue and Aave V3 accounts hold no minimum
// position (no CAP_HELD_REPAY); and minimumDebt reverts, as theirs have none.
func replayNode(t *testing.T, fx *forkFixture, block uint64) *node {
	t.Helper()
	recordedAt := func(sel []byte) bool {
		return lo.SomeBy(lo.Keys(fx.Calls), func(k string) bool {
			f := strings.Fields(k)
			return len(f) == 3 && strings.HasPrefix(f[2], hexutil.Encode(sel))
		})
	}
	actions, caps, minimum := routerABI.Methods["venueActions"].ID, accountABI.Methods["capabilities"].ID,
		accountABI.Methods["minimumDebt"].ID
	older := map[string]bool{string(actions): !recordedAt(actions), string(caps): !recordedAt(caps),
		string(minimum): !recordedAt(minimum)}
	n := newNode(t, block, func(from, to common.Address, data []byte) (bool, []byte) {
		r, ok := fx.Calls[callKey(from, to, data)]
		if !ok && len(data) >= 4 && older[string(data[:4])] {
			if bytes.Equal(data[:4], caps) {
				return true, make([]byte, 32)
			}
			return false, nil
		}
		if !ok {
			t.Errorf("a call the fixture lacks: %s", callKey(from, to, data))
			return false, nil
		}
		return r.OK, common.FromHex(r.Ret)
	}, func(a common.Address) []byte {
		return common.FromHex(fx.Code[strings.ToLower(a.Hex())])
	})
	n.recorded = true
	n.storage = func(a common.Address, slot common.Hash) common.Hash {
		return common.HexToHash(fx.Storage[storageKey(a, slot)])
	}
	return n
}

func storageKey(a common.Address, slot common.Hash) string {
	return strings.ToLower(a.Hex() + " " + slot.Hex())
}

// unstamped is a refresh's Extra less its stamp, the one field that differs from run to run, and, where the fixture
// was recorded before the charge (its ladders carry no cells), less the charge too.
func unstamped(t *testing.T, p entity.Pool, charged bool) string {
	t.Helper()
	e := extraOf(t, p)
	e.LiveAt = 0
	if !charged {
		e.Ladders = lo.Map(e.Ladders, func(l Ladder, _ int) Ladder { return uncharged(l) })
	}
	raw, err := json.Marshal(e)
	require.NoError(t, err)
	return string(raw)
}

// uncharged is a ladder less its charge (gas.go): what a fixture recorded before the charge holds of it.
func uncharged(l Ladder) Ladder {
	l.Cells, l.Second, l.Held, l.Freed, l.OwedOut, l.OwedIn = nil, 0, nil, nil, 0, 0
	l.FirstOwing, l.PostedIn, l.Repay = false, false, false
	return l
}

// recordedCharged reports whether the fixture's pools were recorded with the charge: their ladders carry cells.
func recordedCharged(fx *forkFixture) bool {
	return lo.SomeBy(fx.Pools, func(p entity.Pool) bool {
		var e Extra
		return json.Unmarshal([]byte(p.Extra), &e) == nil && lo.SomeBy(e.Ladders, func(l Ladder) bool {
			return len(l.Cells) != 0
		})
	})
}

// quoting is a recorded pool whose ladders carry no cells, recorded before the charge, given one each, so that a
// simulator quotes it: its tests check amounts, not gas.
func quoting(t *testing.T, p entity.Pool) entity.Pool {
	t.Helper()
	e := extraOf(t, p)
	for k := range e.Ladders {
		if len(e.Ladders[k].Cells) == 0 {
			e.Ladders[k].Cells, e.Ladders[k].Second = quoteCells, quoteSecond
		}
	}
	raw, err := json.Marshal(e)
	require.NoError(t, err)
	p.Extra = string(raw)
	return p
}

// quotingLadder is the same of one ladder.
func quotingLadder(l Ladder) Ladder {
	if len(l.Cells) == 0 {
		l.Cells, l.Second = quoteCells, quoteSecond
	}
	return l
}

// The calm fixtures' fills are checked against the pool to the wei: Base's, and BSC's, a pool on each venue kind the
// chain routes (Aave V3, Venus Core and Lista Moolah, BTCB/USDT, each worth about $100,000), where Lista lends the
// sale's output with nothing owed and its minimum under the sale's first point, so its guard keeps the whole sale.
// The lever fixture's, where leverage-labelled chords meet the rounding of both routes, are checked against the
// tolerance (forkTolerance).
func TestForkReplayMatchesThePoolsSwap(t *testing.T) {
	for _, path := range []string{"testdata/base-fork.json.gz", "testdata/base-fork-lever.json.gz",
		"testdata/bsc-fork.json.gz"} {
		t.Run(strings.TrimSuffix(strings.TrimPrefix(path, "testdata/"), ".json.gz"), func(t *testing.T) {
			fx := loadFixture(t, path)
			sims := replayRefreshes(t, fx)
			if fx.ChainID == uint64(valueobject.ChainIDBSC) { // every venue kind routes both directions
				for _, p := range slices.Concat(fx.Pools, fx.Warm) {
					require.Len(t, extraOf(t, p).Ladders, 2, fx.Kinds[p.Address])
				}
			}
			if len(fx.Swaps) != 0 {
				checkSwaps(t, fx, sims)
			}
			if len(fx.Sweeps) != 0 {
				checkSweeps(t, fx, sims)
			}
		})
	}
}

// replayRefreshes replays the lister and each pool's two refreshes, each at its own block and the second from the
// first's state, through the recorded calls: the same pools and ladders as recorded, one sampler call a refresh. It
// returns each refresh's simulators by pool.
func replayRefreshes(t *testing.T, fx *forkFixture) []map[string]*PoolSimulator {
	t.Helper()
	cfg, charged := fx.Config, recordedCharged(fx)
	pools, _, err := NewPoolsListUpdater(&cfg, replayNode(t, fx, 0).client()).GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, lo.Map(fx.Pools, func(p entity.Pool, _ int) string { return p.Address }),
		lo.Map(pools, func(p entity.Pool, _ int) string { return p.Address }))

	// each pool's two refreshes at its own block, the second from the first's state
	sims := []map[string]*PoolSimulator{{}, {}}
	for k, p := range pools {
		for r, want := range []entity.Pool{fx.Pools[k], fx.Warm[k]} {
			n := replayNode(t, fx, want.BlockNumber)
			got, err := NewPoolTracker(&cfg, n.client()).GetNewPoolState(context.Background(), p,
				pool.GetNewPoolStateParams{})
			require.NoError(t, err)
			requests, sampled := n.counts()
			require.Equal(t, want.Tokens, lo.Map(got.Tokens, func(t *entity.PoolToken, _ int) *entity.PoolToken {
				return &entity.PoolToken{Address: t.Address, Swappable: t.Swappable}
			}))
			require.JSONEq(t, unstamped(t, want, charged), unstamped(t, got, charged), "%s, refresh %d",
				fx.Kinds[p.Address], r)
			require.True(t, lo.EveryBy(extraOf(t, got).Ladders, func(l Ladder) bool { return len(l.Cells) != 0 }),
				"every ladder carries its cells")
			require.Equal(t, want.Reserves, got.Reserves)
			require.Equal(t, want.BlockNumber, got.BlockNumber)
			require.NotEmpty(t, extraOf(t, got).Ladders, fx.Kinds[p.Address])
			s, err := NewPoolSimulator(pool.FactoryParams{EntityPool: got})
			require.NoError(t, err)
			sims[r][got.Address] = s
			probes := recordedProbes(t, fx, n)
			t.Logf("%s, refresh %d: %d requests, %d of them the sampler's, %d probes, ladders of %v points and %v cells",
				fx.Kinds[p.Address], r+1, requests, sampled, probes,
				lo.Map(extraOf(t, got).Ladders, func(l Ladder, _ int) int { return len(l.Points) }),
				lo.Map(extraOf(t, got).Ladders, func(l Ladder, _ int) int { return len(l.Cells) }))
			require.Equal(t, 1, sampled, "one sampler call a refresh")
			p = got
		}
	}
	return sims
}

// checkSwaps checks every fill: exact at a probed size, under the pool and within the haircut, the floors and the
// curvature between two, and refused where the pool refuses.
func checkSwaps(t *testing.T, fx *forkFixture, sims []map[string]*PoolSimulator) {
	cfg := fx.Config
	haircut := new(big.Int).SetUint64(cfg.haircutBps())
	kinds := map[string]bool{}
	type tally struct {
		fills, unquoted, reverts int
		worst                    float64 // bp under the pool between probed sizes, the haircut's share and two units left out
		at                       string  // the worst fill
	}
	tallies := map[string]*tally{}
	for _, sw := range fx.Swaps {
		s := sims[sw.Refresh][sw.Pool]
		require.NotNil(t, s, sw.Pool)
		kind := fx.Kinds[sw.Pool]
		kinds[kind] = true
		label := fmt.Sprintf("%s, refresh %d", kind, sw.Refresh+1)
		if tallies[label] == nil {
			tallies[label] = &tally{}
		}
		ty := tallies[label]
		in, _ := new(big.Int).SetString(sw.AmountIn, 10)
		unused, _ := new(big.Int).SetString(sw.AmountUnused, 10)
		chain, _ := new(big.Int).SetString(sw.AmountOut, 10)
		res, err := s.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: sw.TokenIn, Amount: in}, TokenOut: sw.TokenOut})
		if sw.Reverts {
			require.Error(t, err, "%s at %s: the swap reverts, the simulator quotes", label, sw.AmountIn)
			ty.reverts++
			continue
		}
		if err != nil { // the simulator may quote less than the pool fills, never more: here nothing
			require.False(t, sw.Rung, "%s at %s: a probed size quotes", label, sw.AmountIn)
			require.True(t, errors.Is(err, ladder.ErrNoQuote) || errors.Is(err, ladder.ErrAmountInTooLarge),
				"%s at %s: %v", label, sw.AmountIn, err)
			ty.unquoted++
			continue
		}
		ty.fills++
		require.Zero(t, unused.Cmp(res.RemainingTokenAmountIn.Amount), "%s at %s: the input the pool returns, %s",
			label, sw.AmountIn, res.RemainingTokenAmountIn.Amount)
		if sw.Rung {
			require.Zero(t, chain.Cmp(res.TokenAmountOut.Amount), "%s at %s: the pool pays %s, the simulator %s",
				label, sw.AmountIn, chain, res.TokenAmountOut.Amount)
			continue
		}
		require.LessOrEqual(t, res.TokenAmountOut.Amount.Cmp(chain), 0, "%s at %s: over the pool", label,
			sw.AmountIn)
		short := new(big.Int).Sub(chain, res.TokenAmountOut.Amount)
		short.Sub(short, new(big.Int).Div(new(big.Int).Mul(chain, haircut), big.NewInt(10_000)))
		short.Sub(short, big.NewInt(2))
		bps, _ := new(big.Float).Quo(new(big.Float).SetInt(short), new(big.Float).SetInt(chain)).Float64()
		if bps*10_000 > ty.worst {
			ty.worst, ty.at = bps*10_000, fmt.Sprintf("%s -> %s at %s: pool %s, simulator %s", sw.TokenIn[:6],
				sw.TokenOut[:6], sw.AmountIn, chain, res.TokenAmountOut.Amount)
		}
	}
	labels := slices.Sorted(maps.Keys(tallies))
	for _, label := range labels {
		ty := tallies[label]
		t.Logf("%s: %d fills quoted (exact at every probed size), %d the simulator leaves unquoted, %d the pool "+
			"refuses (and the simulator); between probed sizes at most %.2f bp under the pool past the %d bp haircut "+
			"(%s)", label, ty.fills, ty.unquoted, ty.reverts, ty.worst, cfg.haircutBps(), ty.at)
	}
	for _, label := range labels {
		require.LessOrEqual(t, tallies[label].worst, curvatureBps(fx.ChainID, label), label)
	}
	require.Len(t, kinds, len(lo.Uniq(lo.Values(fx.Kinds))), "fills of every kind")
}

// curvatureBps is what the chord of a concave segment may lose to the pool's curve past the haircut, in bp of the
// output, as measured on the fixtures. A first refresh spends its whole budget on its seeds, about two to one apart
// where these pools price, and refines nothing: its widest loss is a buy's top span, where the price falls 4.4 % on
// Base and less on BSC's pools. A later refresh refines with the probes its seeds leave, which leave a span past the
// 1 bp target.
func curvatureBps(chain uint64, label string) float64 {
	first := strings.HasSuffix(label, "refresh 1")
	if chain == uint64(valueobject.ChainIDBSC) {
		return lo.Ternary(first, 52.4, 4.9)
	}
	return lo.Ternary(first, 152, 3.1)
}

// recordedProbes counts the probes of the sampler's answers the node replayed.
func recordedProbes(t *testing.T, fx *forkFixture, n *node) int {
	t.Helper()
	var probes int
	for _, c := range n.calls() {
		if !c.sampler {
			continue
		}
		var a sampleAnswer
		require.NoError(t, samplerABI.UnpackIntoInterface(&a, "sample",
			common.FromHex(fx.Calls[callKey(c.from, c.to, c.data)].Ret)))
		for _, w := range a.Walks {
			probes += len(w.Probes)
		}
	}
	return probes
}

// A route that crosses one pool twice, against the two swaps the chain ran. The chain charges the second fill of a
// direction its own fee on the book the first left, which the ladder cannot say, so a direction takes one fill per
// route: the simulator refuses a second one with no quote. A fill the other way only moves the pool in the second's
// favour, so it is quoted, and at no more than the chain paid.
func TestForkRouteCrossingOnePoolTwice(t *testing.T) {
	fx := loadFixture(t, "testdata/base-fork.json.gz")
	require.NotEmpty(t, fx.Pairs)
	amount := func(s string) *big.Int {
		a, ok := new(big.Int).SetString(s, 10)
		require.True(t, ok, s)
		return a
	}
	params := func(sw forkSwap) pool.CalcAmountOutParams {
		return pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: sw.TokenIn, Amount: amount(sw.AmountIn)}, TokenOut: sw.TokenOut}
	}
	var refused, otherWay int
	for _, pr := range fx.Pairs {
		label := fmt.Sprintf("%s, %s then %s at %s, %s", fx.Kinds[pr.First.Pool], pr.First.TokenIn[:6],
			pr.Second.TokenIn[:6], pr.First.AmountIn, pr.Second.AmountIn)
		require.False(t, pr.First.Reverts || pr.Second.Reverts, label)
		p, ok := lo.Find(lo.Ternary(pr.Volatile, fx.Volatile, fx.Pools), func(p entity.Pool) bool {
			return p.Address == pr.First.Pool
		})
		require.True(t, ok, label)
		s, err := NewPoolSimulator(pool.FactoryParams{EntityPool: quoting(t, p)})
		require.NoError(t, err)
		first, err := s.CalcAmountOut(params(pr.First))
		require.NoError(t, err, label)
		require.LessOrEqual(t, first.TokenAmountOut.Amount.Cmp(amount(pr.First.AmountOut)), 0, label)
		s.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn:  pool.TokenAmount{Token: pr.First.TokenIn, Amount: amount(pr.First.AmountIn)},
			TokenAmountOut: *first.TokenAmountOut, Fee: *first.Fee, SwapInfo: first.SwapInfo})
		second, err := s.CalcAmountOut(params(pr.Second))
		if pr.First.TokenIn == pr.Second.TokenIn {
			require.ErrorIs(t, err, ladder.ErrNoQuote, label)
			refused++
			continue
		}
		require.NoError(t, err, label)
		require.LessOrEqual(t, second.TokenAmountOut.Amount.Cmp(amount(pr.Second.AmountOut)), 0, label)
		otherWay++
	}
	t.Logf("%d routes: %d second fills of a direction refused, %d the other way quoted at most the chain's", len(fx.Pairs),
		refused, otherWay)
	require.Positive(t, refused)
	require.Positive(t, otherWay)
}

// An output cap cuts a fill. previewSwap then asks again at the input it used, while swap, which the adapter hands the
// whole input, has one pass: that pass pays the ladder's top to the wei and returns the rest, as the simulator says. An
// input cap is no cut: the ladder stops at the cap, the pool fills the cap and returns the rest, and past it the
// simulator quotes nothing.
func TestForkCappedPoolPaysItsTopAndReturnsTheRest(t *testing.T) {
	fx := loadFixture(t, "testdata/base-fork.json.gz")
	require.NotEmpty(t, fx.Capped)
	amount := func(s string) *big.Int {
		a, ok := new(big.Int).SetString(s, 10)
		require.True(t, ok, s)
		return a
	}
	sims := map[string]*PoolSimulator{}
	for _, p := range fx.Capped {
		require.True(t, lo.SomeBy(extraOf(t, p).Ladders, func(l Ladder) bool { return l.Capped }), fx.Kinds[p.Address])
		s, err := NewPoolSimulator(pool.FactoryParams{EntityPool: quoting(t, p)})
		require.NoError(t, err)
		sims[p.Address] = s
	}
	var cut, past, inside int
	for _, sw := range fx.CappedSwaps {
		label := fmt.Sprintf("%s, %s at %s", fx.Kinds[sw.Pool], sw.TokenIn[:6], sw.AmountIn)
		require.False(t, sw.Reverts, label)
		chain, unused := amount(sw.AmountOut), amount(sw.AmountUnused)
		res, err := sims[sw.Pool].CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: sw.TokenIn, Amount: amount(sw.AmountIn)}, TokenOut: sw.TokenOut})
		if err != nil {
			require.ErrorIs(t, err, ladder.ErrAmountInTooLarge, label)
			require.Positive(t, unused.Sign(), label)
			past++
			continue
		}
		require.Zero(t, unused.Cmp(res.RemainingTokenAmountIn.Amount), label)
		if unused.Sign() > 0 || sw.Rung {
			require.Zero(t, chain.Cmp(res.TokenAmountOut.Amount), "%s: the pool pays %s, the simulator %s", label, chain,
				res.TokenAmountOut.Amount)
			if unused.Sign() > 0 {
				cut++
			}
			continue
		}
		require.LessOrEqual(t, res.TokenAmountOut.Amount.Cmp(chain), 0, label)
		inside++
	}
	t.Logf("%d fills cut to the ladder's top, %d past an input's cap the simulator leaves, %d between probes", cut, past,
		inside)
	require.Positive(t, cut)
	require.Positive(t, past)
}
