package efficientprop

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// A listing with a credit mode (doorView's creditMode, 1 to 3) fills inside its pool's custody where that fills it
// whole, and past it through its pool's own swap, its credit route, which may borrow the token it pays out: the
// door's quote answers the cheap fill in the first case and the pool's own preview in the second, and the two differ
// in gas by several times. The front tells them apart itself: its tip-taking quoteFrom, with PLAN_ONLY in the tip
// word, answers a size its plan routes with the route's planned output tagged ROUTED, and a size it fills inside the
// custody with the custody answer untagged (PropAMMFront.quoteFrom). That is the front's own route decision, the one
// its fill takes in the same state, so every sampled size is classed by it in every block it is sampled in.

const (
	// planOnlyBit is the tip word's bit that asks the front's plan (PLAN_ONLY), and routedBit the answer's tag of a
	// size the plan routes (ROUTED).
	planOnlyBit = 253
	routedBit   = 255

	// creditModeAlways is the highest credit mode (CREDIT_ALWAYS): every fill routes.
	creditModeAlways = 3
)

// planOnly is the tip word of a plan read: PLAN_ONLY at no tip, the door's own tip in an eth_call that sets no gas
// price.
var planOnly = new(big.Int).Lsh(big.NewInt(1), planOnlyBit)

// planRead names a listing with a credit mode whose sizes are classed by its front's plan.
type planRead struct {
	front   common.Address
	listing uint32
}

// calls asks the front's plan of every size of both directions, in quoteCalls' order, the door the taker (on which
// the answer does not depend); none where p is nil.
func (p *planRead) calls(door common.Address, tokens [2]common.Address, points [2][]*big.Int) []call {
	if p == nil {
		return nil
	}
	var calls []call
	for dir := range points {
		for _, size := range points[dir] {
			calls = append(calls, call{abi: &frontABI, target: p.front, method: "quoteFrom",
				params: []any{tokens[dir], tokens[1-dir], size, big.NewInt(int64(p.listing)), planOnly, door}})
		}
	}
	return calls
}

// readPlans is the plan calls' answers by direction and size: nil where the call reverted.
func readPlans(rets [][]byte, points [2][]*big.Int) (planned [2][]*big.Int) {
	k := 0
	for dir := range points {
		planned[dir] = make([]*big.Int, len(points[dir]))
		for i := range points[dir] {
			if k < len(rets) {
				planned[dir][i] = unpackUint(&frontABI, "quoteFrom", rets[k])
			}
			k++
		}
	}
	return planned
}

// classify is each size's class by the front's plan (planned) beside the door's quote (outputs): routed where the plan
// answers a nonzero output tagged ROUTED; inside the custody where it answers untagged the door's own output. Any other
// size is no point (its output nil): the door refused it, the plan refused it (0, or a revert), or the plan answered
// another amount untagged, which no state of the front gives beside the door's quote.
func classify(outputs, planned [2][]*big.Int) (classed [2][]*big.Int, routed [2][]bool) {
	for dir := range outputs {
		classed[dir] = make([]*big.Int, len(outputs[dir]))
		routed[dir] = make([]bool, len(outputs[dir]))
		for i, out := range outputs[dir] {
			if out == nil || i >= len(planned[dir]) || planned[dir][i] == nil {
				continue
			}
			pl := planned[dir][i]
			switch {
			case pl.Bit(routedBit) == 1:
				if new(big.Int).SetBit(pl, routedBit, 0).Sign() > 0 {
					classed[dir][i], routed[dir][i] = out, true
				}
			case pl.Cmp(out) == 0:
				classed[dir][i] = out
			}
		}
	}
	return classed, routed
}

// collectClassedRun is collectRun on sizes classed by the front's plan (routed; nil where they are not, all inside the
// custody): its run also ends at a size inside the custody after a routed one, so a ladder is custody points, then
// routed points, and from is the index of its first routed point, the run's length where none routes. A falling
// output ends the run there as anywhere: a route whose pool fees or minimum pay less than the custody's last point
// is not quoted past it.
func collectClassedRun(sizes, outputs []*big.Int, ok, routed []bool) (run []Point, from int) {
	from = -1
	for i, size := range sizes {
		pass := i < len(ok) && ok[i] && i < len(outputs) && outputs[i] != nil && outputs[i].Sign() > 0 &&
			outputs[i].BitLen() <= maxPointBits && size.Sign() > 0 && size.BitLen() <= maxPointBits
		if !pass {
			if len(run) > 0 {
				break
			}
			continue
		}
		r := i < len(routed) && routed[i]
		if from >= 0 && !r {
			break
		}
		x, y := uint256.MustFromBig(size), uint256.MustFromBig(outputs[i])
		if len(run) > 0 && (!x.Gt(run[len(run)-1][0]) || y.Lt(run[len(run)-1][1])) {
			break
		}
		if r && from < 0 {
			from = len(run)
		}
		run = append(run, Point{x, y})
		if len(run) == maxLadderPoints {
			break
		}
	}
	if from < 0 {
		from = len(run)
	}
	return run, from
}

// creditOf is a quoted credit listing's Credit: its ladders, each one's first routed point (from) and each direction's
// gas by its input, the pair's base or its quote (baseToken1: the base is token 1).
func creditOf(cl *CreditListing, baseToken1 bool, from [2]int, ladders [2][]Point) *Credit {
	byDir := func(baseIn, quoteIn int64) [2]int64 {
		if baseToken1 {
			return [2]int64{quoteIn, baseIn}
		}
		return [2]int64{baseIn, quoteIn}
	}
	custody, route := byDir(cl.CustodyBaseIn, cl.CustodyQuoteIn), byDir(cl.RouteBaseIn, cl.RouteQuoteIn)
	c := &Credit{Ladders: ladders, From: from, Gas: [2][2]int64{{custody[0], route[0]}, {custody[1], route[1]}}}
	if limit := byDir(cl.RouteLimitBaseIn, cl.RouteLimitQuoteIn); limit != [2]int64{} {
		c.Limit = &limit
	}
	return c
}

// gridReserve is the reserve of token out a direction's sampling grid is scaled by (probeSizes): its pool's balance,
// or on a listing with a credit mode (credit) the reserve the last state recorded, so the grid follows the last ladder
// unscaled where it has one: a pool that may borrow the token pays out no amount its balance bounds.
func gridReserve(p entity.Pool, out int, balance *big.Int, credit bool) *big.Int {
	if credit && len(p.Reserves) == 2 {
		if r, ok := new(big.Int).SetString(p.Reserves[out], 10); ok && r.Sign() > 0 {
			return r
		}
	}
	return balance
}
