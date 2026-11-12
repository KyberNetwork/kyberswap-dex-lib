package everlongflamm

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync/atomic"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

func strs(v []*big.Int) []string {
	return lo.Map(v, func(a *big.Int, _ int) string { return a.String() })
}

func pt(x, y uint64) Point { return Point{uint256.NewInt(x), uint256.NewInt(y)} }

func TestFirstSeedsFollowTheInputDecimals(t *testing.T) {
	for d, want := range map[int][2]string{
		6:  {"1000", "1000000000000000"},                         // 0.001 to a billion USDC
		8:  {"100000", "100000000000000000"},                     // 0.001 to a billion cbBTC
		18: {"1000000000000000", "1000000000000000000000000000"}, // 0.001 to a billion WETH
		0:  {"1", "1000000000"},                                  // a unit under three decimals
	} {
		s := seedsOf(d, nil, nil, firstRefreshSeeds, defaultRungs)
		require.Len(t, s, firstRefreshSeeds, "decimals %d", d)
		require.Equal(t, want, [2]string{s[0].String(), s[len(s)-1].String()}, "decimals %d", d)
		for k := 1; k < len(s); k++ {
			require.Positive(t, s[k].Cmp(s[k-1]), "ascending and distinct")
		}
	}
	require.Equal(t, []string{"500"}, strs(seedsOf(6, big.NewInt(500), nil, firstRefreshSeeds, defaultRungs)),
		"a notional cap under the floor: the cap alone")
	s := seedsOf(8, big.NewInt(300_000_000), nil, firstRefreshSeeds, defaultRungs)
	require.Equal(t, "300000000", s[len(s)-1].String(), "the cap ends them")
	require.Len(t, seedsOf(8, nil, nil, 21, 20), 21, "a direction's share of the pool's budget")
}

func TestSteadySeedsRunFromUnderTheLastFirstPointToEightTimesTheLastTop(t *testing.T) {
	last := &Ladder{Points: []Point{pt(1_000_000, 1_000_000_000), pt(50_000_000, 49_000_000_000)}}
	s := seedsOf(8, nil, last, firstRefreshSeeds, defaultRungs)
	require.Len(t, s, 1+defaultRungs)
	require.Equal(t, []string{"125000", "1000000"}, strs(s[:2]), "an eighth of the first point, then the point")
	require.Equal(t, "400000000", s[len(s)-1].String(), "eight times the top")

	last.Points[0] = pt(100_000, 100_000_000) // at the floor: no size under it
	s = seedsOf(8, nil, last, firstRefreshSeeds, defaultRungs)
	require.Len(t, s, defaultRungs)
	require.Equal(t, "100000", s[0].String())

	last.Points[0] = pt(400_000, 400_000_000) // an eighth under the floor: the floor
	require.Equal(t, "100000", seedsOf(8, nil, last, firstRefreshSeeds, defaultRungs)[0].String())

	s = seedsOf(8, big.NewInt(200_000_000), last, firstRefreshSeeds, defaultRungs)
	require.Equal(t, "200000000", s[len(s)-1].String(), "the cap bounds the steady range too")

	dusty := &Ladder{Points: []Point{pt(1_000, 9_999)}} // its top paid under dustOut: no last ladder
	require.Equal(t, strs(seedsOf(8, nil, nil, firstRefreshSeeds, defaultRungs)),
		strs(seedsOf(8, nil, dusty, firstRefreshSeeds, defaultRungs)))
	require.Equal(t, strs(seedsOf(8, nil, nil, firstRefreshSeeds, defaultRungs)),
		strs(seedsOf(8, nil, &Ladder{}, firstRefreshSeeds, defaultRungs)))
}

// A feed move raises a direction's first point: the pool takes no sale under 2 cbBTC for one refresh. Nothing in the
// steady range would bring it back down, so each refresh seeds one size an eighth of the last first point, and the
// first point falls eight times a refresh to the input's floor.
func TestTrackerLowersAFirstPointARefreshRaised(t *testing.T) {
	w, cfg := testWorld(), testConfig()
	fp := w.pools[common.HexToAddress(poolX)]
	quote := fp.quote
	var moving atomic.Bool
	moving.Store(true)
	fp.quote = func(from common.Address, i, j int, a *big.Int) (*big.Int, *big.Int, bool) {
		if moving.Load() && i == 0 && a.Cmp(big.NewInt(200_000_000)) < 0 {
			return nil, nil, false
		}
		return quote(from, i, j, a)
	}
	n, pools := listed(t, w, cfg)
	tr := NewPoolTracker(cfg, n.client())
	firstOf := func(p entity.Pool) *big.Int {
		l, ok := lo.Find(extraOf(t, p).Ladders, func(l Ladder) bool { return l.In == 0 })
		require.True(t, ok)
		return l.Points[0][0].ToBig()
	}
	p, err := refresh(tr, pools[0])
	require.NoError(t, err)
	want := firstOf(p)
	require.GreaterOrEqual(t, want.Cmp(big.NewInt(200_000_000)), 0)
	moving.Store(false)
	floor := pow10(8 - 3)
	var falls int
	for want.Cmp(floor) > 0 {
		require.Less(t, falls, 8, "the first point reaches the floor")
		p, err = refresh(tr, p)
		require.NoError(t, err)
		want = maxBig(new(big.Int).Rsh(want, 3), floor)
		require.Equal(t, want, firstOf(p), "refresh %d after the move", falls+1)
		falls++
	}
	t.Logf("the first point back at the floor after %d refreshes", falls)
	require.Equal(t, 4, falls)
}

// Every direction a pool routes shares its budget of maxProbesPerRefresh probes, at most maxProbesPerDirection each,
// the hook pair first; past maxSampledDirections the rest are not routed this refresh.
func TestTrackerSplitsThePoolsProbeBudget(t *testing.T) {
	for name, tc := range map[string]struct {
		tokens         int
		every          bool // every pair trades; else testWorld's three directions
		routed, budget int
	}{
		"three directions": {tokens: 3, routed: 3, budget: 21},
		"six":              {tokens: 3, every: true, routed: 6, budget: 10},
		"twenty":           {tokens: 5, every: true, routed: maxSampledDirections, budget: 4},
	} {
		t.Run(name, func(t *testing.T) {
			w, cfg := testWorld(), testConfig()
			fp := w.pools[common.HexToAddress(poolX)]
			for k := len(fp.tokens); k < tc.tokens; k++ {
				fp.tokens = append(fp.tokens, tokenCfg(fmt.Sprintf("0x%040x", 0xe0+k), 6))
			}
			if tc.every {
				fp.tradable = new(big.Int)
				for i := range tc.tokens {
					for j := range tc.tokens {
						if i != j {
							fp.tradable.SetBit(fp.tradable, 16*i+j, 1)
						}
					}
				}
				fp.quote = oracle(1, 1, 5, 100_000_000_000)
			}
			// every pair the mask names, stepped: the everlong shape routes the hook pair alone, so the directions
			// are set here as a pricer of the step rule would have them, the hook pair first
			n, _ := listed(t, w, cfg)
			tr := NewPoolTracker(cfg, n.client())
			st, err := tr.read(context.Background(), poolX, factoryA, len(fp.tokens), nil)
			require.NoError(t, err)
			var dirs []direction
			for _, pair := range [][2]int{{0, 1}, {1, 0}} {
				if fp.tradable.Bit(16*pair[0]+pair[1]) == 1 {
					dirs = append(dirs, direction{in: pair[0], out: pair[1], hook: fp.hook, rule: ruleStep,
						cells: []Cell{{Gas: 1}}})
				}
			}
			for i := range tc.tokens {
				for j := range tc.tokens {
					if i != j && max(i, j) > 1 && fp.tradable.Bit(16*i+j) == 1 {
						dirs = append(dirs, direction{in: i, out: j, hook: fp.hook, rule: ruleStep,
							cells: []Cell{{Gas: 1}}})
					}
				}
			}
			ladders, err := tr.ladders(context.Background(), poolX, st, dirs, &Extra{}, nil)
			require.NoError(t, err)
			require.Len(t, ladders, tc.routed)
			asked := n.asked(t, 0)
			require.Len(t, asked, 1)
			require.Len(t, asked[0].asks, tc.routed)
			for k, a := range asked[0].asks {
				require.Equal(t, int64(tc.budget), a.MaxProbes.Int64())
				require.Len(t, a.Seeds, tc.budget, "a first refresh's seeds, thinned to the budget")
				in, out := fp.index(a.TokenIn), fp.index(a.TokenOut)
				require.Equal(t, k < 2, in < 2 && out < 2, "the hook pair first")
				require.Zero(t, a.TierA.Sign(), "no row, no tier")
			}
		})
	}
}

func TestBuildEndsAtRefusalsFallsAndCeilings(t *testing.T) {
	st := &poolState{tokens: []tokenConfigResult{{Config: tokenCfg(cbBTC, 8)}, {Config: tokenCfg(usdc, 6)}}}
	ladderAt := func(probes [][4]int64) Ladder {
		var w walk
		for _, p := range probes {
			ask, used, out := big.NewInt(p[0]), big.NewInt(p[1]), big.NewInt(p[2])
			w.probes = append(w.probes, sampled{ask: ask, used: used, out: out, fee: new(big.Int),
				status: lo.Ternary(p[3] == 1, fillStatus(ask, used, out), statusRefused)})
		}
		l, _ := build(direction{in: 0, out: 1, cells: []Cell{{Gas: 7}}}, st, w, w.words)
		return l
	}
	// a refusal ends the ladder though a larger size priced, once a price pays at least dustOut
	l := ladderAt([][4]int64{{100, 100, 99_000, 1}, {200, 200, 197_000, 1}, {400, 0, 0, 0}, {800, 800, 780_000, 1}})
	require.Len(t, l.Points, 2)
	require.False(t, l.Capped)
	require.Equal(t, []Cell{{Gas: 7}}, l.Cells)

	// an output that falls ends it
	l = ladderAt([][4]int64{{100, 100, 99, 1}, {200, 200, 98, 1}})
	require.Len(t, l.Points, 1)

	// a cut past a rounding is the ceiling
	l = ladderAt([][4]int64{{100, 100, 99, 1}, {200, 150, 148, 1}, {400, 150, 148, 1}})
	require.Len(t, l.Points, 2)
	require.True(t, l.Capped)
	require.Equal(t, "150", l.Points[1][0].String())

	// a rounding cut is not, while the next size uses more
	l = ladderAt([][4]int64{{100_000, 99_999, 99, 1}, {200_000, 199_999, 197, 1}, {400_000, 399_999, 390, 1}})
	require.Len(t, l.Points, 3)
	require.False(t, l.Capped)

	// a rounding cut whose next size uses no more is the ceiling
	l = ladderAt([][4]int64{{100_000, 99_999, 99, 1}, {200_000, 99_999, 99, 1}})
	require.Len(t, l.Points, 1)
	require.True(t, l.Capped)

	// nothing priced: nothing stands
	_, ok := build(direction{in: 0, out: 1}, st, walk{probes: []sampled{{ask: big.NewInt(1), used: new(big.Int),
		out: new(big.Int), fee: new(big.Int), status: statusRefused}}}, pairWords{})
	require.False(t, ok)
}

// A sampler probe is read as the sampler reads it, and a status it does not give is malformed.
func TestSamplerProbeStatuses(t *testing.T) {
	p := func(ask, used, out, status int64) sampleProbe {
		return sampleProbe{Ask: big.NewInt(ask), Used: big.NewInt(used), Out: big.NewInt(out), FeeWad: new(big.Int),
			GasUsed: new(big.Int), Status: big.NewInt(status)}
	}
	for name, tc := range map[string]struct {
		p    sampleProbe
		want probeStatus
	}{
		"priced":                     {p(100, 100, 99, 1), statusPriced},
		"clipped":                    {p(100, 50, 49, 2), statusClipped},
		"refused":                    {p(100, 0, 0, 3), statusRefused},
		"starved":                    {p(100, 0, 0, 4), statusStarved},
		"malformed":                  {p(100, 0, 0, 5), statusMalformed},
		"no status":                  {p(100, 100, 99, 0), statusMalformed},
		"a status it does not give":  {p(100, 100, 99, 6), statusMalformed},
		"priced, more used than ask": {p(100, 101, 99, 1), statusMalformed},
		"priced, nothing paid":       {p(100, 100, 0, 1), statusMalformed},
		"a status past a byte":       {sampleProbe{Ask: big.NewInt(1), Used: big.NewInt(1), Out: big.NewInt(1), FeeWad: new(big.Int), GasUsed: new(big.Int), Status: big.NewInt(257)}, statusMalformed},
	} {
		require.Equal(t, tc.want, statusOf(tc.p), name)
	}
}

// testRPCError is a node's JSON-RPC error with its code and, where set, its data.
type testRPCError struct {
	code int
	msg  string
	data any
}

func (e *testRPCError) Error() string  { return e.msg }
func (e *testRPCError) ErrorCode() int { return e.code }
func (e *testRPCError) ErrorData() any { return e.data }

func TestClassifyTakesTheFirstListThatMatches(t *testing.T) {
	code := func(c int, msg string) error { return &testRPCError{code: c, msg: msg} }
	for name, tc := range map[string]struct {
		err   error
		want  nodeClass
		retry bool
	}{
		"code 3":                         {code(3, "execution reverted"), classNotRun, true},
		"revert data under another code": {&testRPCError{code: -32000, msg: "error", data: "0xf902f96b"}, classNotRun, true},
		"a revert's words, any case":     {errors.New("VM Exception: Reverted"), classNotRun, true},
		"-32601":                         {code(-32601, "method not found"), classNotRun, false},
		"-32602":                         {code(-32602, "too many arguments, want at most 2"), classNotRun, false},
		"-32604":                         {code(-32604, "method not supported"), classNotRun, false},
		"-32600":                         {code(-32600, "invalid request"), classNotRun, false},
		"-32600 with a rate limit":       {code(-32600, "Rate limit exceeded"), classSkipped, false},
		"an override":                    {code(-32000, "State override not supported"), classNotRun, false},
		"gas over the node's cap":        {code(-32000, "gas limit too high"), classNotRun, false},
		"gas allowance":                  {code(-32000, "gas required exceeds allowance (25000000)"), classNotRun, false},
		"gas limit reached, a rate code": {code(-32005, "Gas limit reached"), classNotRun, false},
		"the call out of gas":            {code(-32000, "out of gas"), classMissed, false},
		"HTTP 413":                       {rpc.HTTPError{StatusCode: 413, Status: "413 Request Entity Too Large"}, classNotRun, false},
		"HTTP 429":                       {rpc.HTTPError{StatusCode: 429, Status: "429 Too Many Requests"}, classSkipped, false},
		"HTTP 503":                       {rpc.HTTPError{StatusCode: 503, Status: "503 Service Unavailable"}, classSkipped, false},
		"HTTP 502":                       {rpc.HTTPError{StatusCode: 502, Status: "502 Bad Gateway"}, classMissed, false},
		"a rate code":                    {code(-32005, "busy"), classSkipped, false},
		"a rate limit's words":           {errors.New("Too Many Requests"), classSkipped, false},
		"credits":                        {code(-32000, "Your app has exceeded its compute units: credits limited"), classSkipped, false},
		"a null answer":                  {errNullAnswer, classSkipped, false},
		"an answer not the sampler's":    {fmt.Errorf("%w: tag 01020304", errNotSampled), classNotRun, true},
		"header not found":               {code(-32000, "header not found"), classBlockMissed, false},
		"missing trie node":              {code(-32000, "missing trie node abc"), classBlockMissed, false},
		"the deadline":                   {context.DeadlineExceeded, classMissed, false},
		"the deadline, a rate's words":   {fmt.Errorf("rate limit: %w", context.DeadlineExceeded), classMissed, false},
		"cancelled":                      {context.Canceled, classMissed, false},
		"a transport error":              {errors.New("dial tcp: connection refused"), classMissed, false},
		"a node's own error":             {code(-32603, "internal error"), classMissed, false},
	} {
		require.Equal(t, tc.want, classify(tc.err), name)
		require.Equal(t, tc.want == classNotRun, samplerUnsupported(tc.err), name)
		require.Equal(t, tc.retry, retryable(tc.err), name)
	}
}

func TestGeometricSizes(t *testing.T) {
	require.Equal(t, []string{"1", "10", "100", "1000"}, strs(geometric(big.NewInt(1), big.NewInt(1000), 4)))
	require.Equal(t, []string{"5"}, strs(geometric(big.NewInt(5), big.NewInt(5), 4)))
	require.Equal(t, []string{"1", "2"}, strs(geometric(big.NewInt(1), big.NewInt(2), 6)), "distinct sizes only")
}

func TestFamilyHashesTheCodeWithItsImmutablesZeroed(t *testing.T) {
	code := []byte{0x60, 0x80, 0xaa, 0xbb, 0xcc, 0x60, 0x40}
	masked := []byte{0x60, 0x80, 0, 0, 0, 0x60, 0x40}
	families := []HookFamily{
		{Name: "other", CodeHash: crypto.Keccak256Hash([]byte{1}).Hex()},
		{Name: "hook", CodeHash: crypto.Keccak256Hash(masked).Hex(), Immutables: [][2]int{{2, 3}}},
	}
	require.Equal(t, 1, familyOf(families, code))
	require.Equal(t, 1, familyOf(families, []byte{0x60, 0x80, 1, 2, 3, 0x60, 0x40}), "another deployment's immutables")
	require.Equal(t, -1, familyOf(families, []byte{0x60, 0x81, 0xaa, 0xbb, 0xcc, 0x60, 0x40}), "other code")
	require.Equal(t, -1, familyOf(families, nil))
	families[1].Immutables = [][2]int{{5, 9}}
	require.Equal(t, -1, familyOf(families, code), "a range past the code")
}

func TestPairBitAndLaneVersion(t *testing.T) {
	require.Equal(t, 160, pairBit(0, 1))
	require.Equal(t, 161, pairBit(2, 0))
	require.Equal(t, 162, pairBit(1, 2))
	require.Equal(t, 250, pairBit(13, 12))
	require.Equal(t, uint64(0), laneVersion(new(big.Int).Lsh(big.NewInt(1), 96)))
	require.Equal(t, uint64(1), laneVersion(new(big.Int).Lsh(big.NewInt(1), 97)))
	require.Equal(t, uint64(1), laneVersion(new(big.Int).Or(new(big.Int).Lsh(big.NewInt(1), 97),
		new(big.Int).Lsh(big.NewInt(1), 104))), "the pause bits above it are not the version")
}

func TestSingleTupleOutputsDecodeOnlyWrapped(t *testing.T) {
	cfg := tokenConfig{Token: common.HexToAddress(usdc), Decimals: 6, MaxSwapNotional: big.NewInt(1),
		ReserveTarget: big.NewInt(2), MinSupplyBatch: big.NewInt(3), Held: big.NewInt(4), DebtFree: true}
	data, err := poolABI.Methods["tokenConfig"].Outputs.Pack(cfg)
	require.NoError(t, err)

	var wrapped tokenConfigResult
	require.NoError(t, poolABI.UnpackIntoInterface(&wrapped, "tokenConfig", data))
	require.Equal(t, cfg, wrapped.Config)

	var bare tokenConfig
	require.Panics(t, func() { _ = poolABI.UnpackIntoInterface(&bare, "tokenConfig", data) },
		"go-ethereum copies a single output into the target's first field")
}
