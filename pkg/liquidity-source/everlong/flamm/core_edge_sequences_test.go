package everlongflamm

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

type coreEdgeSeqRow struct {
	K     string          `json:"k"`
	Seq   string          `json:"seq"`
	I     int             `json:"i"`
	Ts    uint64          `json:"ts"`
	S     json.RawMessage `json:"s"`
	Lever bool            `json:"lever"`
	D     int             `json:"d"`
	A     uint256.Int     `json:"a"`
	M     uint256.Int     `json:"m"`
	Late  uint64          `json:"late"`
	R     *string         `json:"r"`
	E     *string         `json:"e"`
	Ev    []struct {
		T    string `json:"t"`
		Data string `json:"data"`
	} `json:"ev"`
	Post json.RawMessage `json:"post"`
	Op   string          `json:"op"`
	Ok   bool            `json:"ok"`
}

// coreEdgeSeqFixtures: executed sequences (testdata/gen/CoreEdgeSeq.t.sol).
var coreEdgeSeqFixtures = map[string]string{
	"51302915": "e6e14db48bcfe28a3f87ff072a5cc2ac70e92e4a27d669c39d3dec9338fca99c",
	"51324800": "53414398f523e1507ed8fdd2c2e4f91642f02dba22a195068af4a5ac948b553d",
	"51326000": "1906c0dfc12c37b6f3d7dd5b52ffee245cf1107b6aa5da43ea29dc0773ab341e",
}

var (
	coreEdgeCbbtc = common.HexToAddress("0xcbB7C0000aB88B473b1f5aFd9ef808440eed33Bf")
	coreEdgeUsdc  = common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913")
)

// coreEdgeReSkip names the fields a non-core move may change (a tracker re-read); every other field must already
// equal the port's carried state.
func coreEdgeReSkip(op string) map[string]bool {
	m := map[string]bool{}
	add := func(p ...string) {
		for _, x := range p {
			m[x] = true
		}
	}
	switch op {
	case "setSpread", "setMaxSpreadAge":
		add("s.Hooks.Spread")
	case "setLoanConfig":
		for _, f := range []string{"SwapPriceBandWad", "FeeFloorWad", "MaxSwapNotional", "ReserveTarget"} {
			add("s.Pool.Loans[0]." + f)
		}
	case "setFeeBounds":
		add("s.FeeFloorWad", "s.FeeCapWad")
	case "setPaused":
		add("s.Paused")
	case "setLevPaused":
		add("s.LevPaused")
	case "setFeatures":
		add("s.Pool.Features")
	case "setDials":
		add("s.Pool.LtvWad", "s.Pool.PhiWad", "s.Router.PinLtvWad")
	case "setVenueCaps":
		add("s.Router.Venues[0].DebtCap", "s.Router.Venues[0].SupplyCap", "s.Router.Venues[0].MaxBorrowRateWad")
	case "setVenueFlags":
		add("s.Router.Venues[0].BorrowEnabled", "s.Router.Venues[0].SupplyEnabled")
	case "feeds", "sequencer":
		add("s.Feed")
	case "irm":
		add("s.Router.Venues[0].Morpho.IrmReadable")
	case "oracle":
		add("s.Router.Venues[0].Morpho.OracleOk", "s.Router.Venues[0].Morpho.OraclePrice",
			"s.Router.Venues[0].Morpho.OracleZero")
	case "morpho":
		add("s.Router.Venues[0].Morpho.Market", "s.Router.Venues[0].Morpho.RateAtTarget")
	case "loanCaps":
		add("s.Router.Loans[0].DebtCap", "s.Router.Loans[0].SupplyCap", "s.Router.Loans[0].BorrowEnabled")
	case "donate":
		add("s.Router.Venues[0].Morpho.Market", "s.Router.Venues[0].Morpho.RateAtTarget",
			"s.Router.Venues[0].Morpho.Position")
	case "warpquiet":
	case "rewind", "deposit":
		return nil
	default:
		panic("unknown re-read op " + op)
	}
	return m
}

func TestCoreEdgeSequences(t *testing.T) {
	t.Parallel()
	blocks := make([]string, 0, len(coreEdgeSeqFixtures))
	for b := range coreEdgeSeqFixtures {
		blocks = append(blocks, b)
	}
	sort.Strings(blocks)
	for _, blk := range blocks {
		t.Run(blk, func(t *testing.T) {
			rep := &coreEdgeReport{t: t}
			executed := coreEdgeReplaySeq(t, rep, blk, nil)
			t.Logf("block %s: %d checks, %d mismatches, executed %v", blk, rep.checks, rep.fails, executed)
		})
	}
}

// coreEdgeReplaySeq replays one block's sequences from the port's own post-states; mutate (sensitivity runs) may
// perturb every post-state the port carries forward.
func coreEdgeReplaySeq(t *testing.T, rep *coreEdgeReport, blk string, mutate func(s *flammState)) map[string]int {
	path, digest := coreEdgeFixturePath("seq", blk), coreEdgeSeqFixtures[blk]
	var s *flammState
	var now uint64
	executed := map[string]int{}
	for _, line := range coreEdgeLoad(t, path, digest) {
		if rep.quiet && rep.fails > 0 {
			break
		}
		var row coreEdgeSeqRow
		require.NoError(t, json.Unmarshal(line, &row))
		where := fmt.Sprintf("%s/%s#%d", blk, row.Seq, row.I)
		switch row.K {
		case "begin":
			var reads *flammReads
			s, reads = coreEdgeState(t, row.S)
			now = reads.Timestamp
			continue
		case "pv":
			require.Equal(t, now, row.Ts, where)
			ret, rev := coreEdgeStr(row.R), coreEdgeStr(row.E)
			var got []uint256.Int
			var err error
			before := s.clone()
			if row.Lever {
				got, err = coreEdgeLeverWords(s, row.D == 1, &row.A, now)
			} else {
				got, err = coreEdgeSwapWords(s, row.D == 1, &row.A, now)
			}
			rep.match(fmt.Sprintf("%s preview lever=%v d=%d a=%s", where, row.Lever, row.D, row.A.Dec()),
				row.R != nil, ret, rev, err, got)
			if d := coreEdgeDiff(s, before, nil); len(d) != 0 {
				rep.fail("%s: preview mutated the state: %v", where, d)
			}
			continue
		case "warp":
			now = row.Ts
			want, _ := coreEdgeState(t, row.Post)
			rep.checks++
			if d := coreEdgeDiff(s, want, nil); len(d) != 0 {
				rep.fail("%s warp: reads moved without a transaction:\n  %s", where, strings.Join(d, "\n  "))
			}
			continue
		case "re":
			now = row.Ts
			want, _ := coreEdgeState(t, row.Post)
			rep.checks++
			if skip := coreEdgeReSkip(row.Op); skip == nil {
				// a snapshot rewind: nothing to carry
			} else if d := coreEdgeDiff(s, want, skip); len(d) != 0 {
				rep.fail("%s re-read %s: carried state drifted:\n  %s", where, row.Op, strings.Join(d, "\n  "))
			}
			s = want
			continue
		case "x":
		default:
			t.Fatalf("row kind %q", row.K)
		}
		now = row.Ts
		want, _ := coreEdgeState(t, row.Post)
		label := fmt.Sprintf("%s exec lever=%v d=%d a=%s min=%s", where, row.Lever, row.D, row.A.Dec(), row.M.Dec())
		var post *flammState
		pre := s.clone()
		var words, evw []uint256.Int
		evName := ""
		var err error
		if row.Lever {
			var r *leverResult
			r, post, err = s.executeLever(row.D == 1, &row.A, &row.M, now-row.Late, now)
			if err == nil {
				words = []uint256.Int{r.AmountInUsed, r.AmountOut}
				evw = []uint256.Int{r.AmountInUsed, r.AmountOut, r.SpreadPpm, r.CrAfterWad}
				evName = map[bool]string{true: "LeverUp", false: "LeverDown"}[r.Up]
			}
		} else {
			tin, tout := coreEdgeCbbtc, coreEdgeUsdc
			if row.D == 0 {
				tin, tout = tout, tin
			}
			var r *swapResult
			r, post, err = s.executeSwap(tin, tout, &row.A, &row.M, now-row.Late, now)
			if err == nil {
				words = []uint256.Int{r.AmountInUsed, r.AmountOut}
				evw = []uint256.Int{e2eWord(r.PoolAssetIn), r.AmountInUsed, r.AmountOut, r.FeeOut, r.FeeWad,
					r.SpotAfterWad}
				evName = "Swap"
			}
		}
		rep.match(label, row.R != nil, coreEdgeStr(row.R), coreEdgeStr(row.E), err, words)
		rep.checks++
		if d := coreEdgeDiff(s, pre, nil); len(d) != 0 {
			rep.fail("%s: the execution wrote its receiver:\n  %s", label, strings.Join(d, "\n  "))
		}
		if row.R == nil {
			rep.checks++
			if len(row.Ev) != 0 {
				rep.fail("%s: reverted call emitted %v", label, row.Ev)
			}
			if d := coreEdgeDiff(s, want, nil); len(d) != 0 {
				rep.fail("%s: chain state moved on a revert:\n  %s", label, strings.Join(d, "\n  "))
			}
			continue
		}
		if err != nil {
			s = want
			continue
		}
		executed[evName]++
		for _, tag := range coreEdgePaths(s, want) {
			executed[evName+" "+tag]++
		}
		rep.checks++
		if len(row.Ev) != 1 || row.Ev[0].T != evName {
			rep.fail("%s: chain events %v, go %s", label, row.Ev, evName)
		} else if cw := coreEdgeWords(row.Ev[0].Data); fmt.Sprint(cw) != fmt.Sprint(evw) {
			rep.fail("%s: event chain %v go %v", label, cw, evw)
		}
		rep.checks++
		if d := coreEdgeDiff(post, want, nil); len(d) != 0 {
			rep.fail("%s: post-state differs:\n  %s", label, strings.Join(d, "\n  "))
			post = want
		}
		if mutate != nil {
			mutate(post)
		}
		s = post
	}
	return executed
}

// TestCoreEdgeSequenceSensitivity: carrying a Morpho accrual stamp one second stale must break the replay.
func TestCoreEdgeSequenceSensitivity(t *testing.T) {
	t.Parallel()
	rep := &coreEdgeReport{t: t, quiet: true}
	coreEdgeReplaySeq(t, rep, "51324800", func(s *flammState) {
		m := &s.Router.Venues[0].Morpho.Market
		m.LastUpdate.SubUint64(&m.LastUpdate, 1)
	})
	require.NotZero(t, rep.fails, "a stale accrual stamp changed no step")
}

func coreEdgeStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// coreEdgePaths labels which settlement legs an executed step moved, from the pre and post states.
func coreEdgePaths(pre, post *flammState) []string {
	var tags []string
	a, b := &pre.Router.Venues[0], &post.Router.Venues[0]
	cmp := func(name string, x, y *uint256.Int) {
		switch {
		case y.Gt(x):
			tags = append(tags, name+"+")
		case y.Lt(x):
			tags = append(tags, name+"-")
		}
	}
	cmp("borrowShares", &a.Morpho.Position.BorrowShares, &b.Morpho.Position.BorrowShares)
	cmp("collateral", &a.Morpho.Position.Collateral, &b.Morpho.Position.Collateral)
	cmp("supplyShares", &a.Morpho.Position.SupplyShares, &b.Morpho.Position.SupplyShares)
	cmp("liquid", &pre.Pool.Loans[0].Liquid, &post.Pool.Loans[0].Liquid)
	if !post.LastLeverSpreadPpm.Eq(&pre.LastLeverSpreadPpm) {
		tags = append(tags, "spreadStored")
	}
	if post.Router.Venues[0].Morpho.Market.LastUpdate.Gt(&pre.Router.Venues[0].Morpho.Market.LastUpdate) {
		tags = append(tags, "accrued")
	}
	return tags
}
