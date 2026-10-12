package lotflow

import (
	"fmt"
	"math/big"
	"strings"
)

// World carries simulated state from one swap to the next, across venues: constituent pools are
// shared by source id (two baskets holding AMZN trade the same AMZN pools), venues by LOT. It is the
// simulator-side counterpart of dex-lib's CloneState/UpdateBalance for a sequence of swaps.
type World struct {
	pools  map[string]*LegPool
	venues map[string]*venueSnap
}

type venueSnap struct {
	sqrtP    *big.Int
	tick     int
	standing *StandingPosition
	pmLot    *big.Int
}

func NewWorld() *World {
	return &World{pools: map[string]*LegPool{}, venues: map[string]*venueSnap{}}
}

// Absorb records the state a successful venue swap on m left behind.
func (w *World) Absorb(m *Market, res *SwapResult) {
	for _, ps := range res.post {
		for id, p := range ps {
			w.pools[strings.ToLower(id)] = p
		}
	}
	if m.Mq != nil && res.convPost != nil {
		w.pools[strings.ToLower(m.Mq.ConvPool)] = res.convPost
	}
	var st *StandingPosition
	if res.Standing != nil {
		st = res.Standing.After
	} else {
		st = m.Venue.Standing
	}
	w.venues[strings.ToLower(m.Lot)] = &venueSnap{sqrtP: res.PostSqrtPriceX96, tick: res.PostTick, standing: st,
		pmLot: res.PmLotAfter}
}

// AbsorbPool records a plain swap's effect on one constituent pool of m.
func (w *World) AbsorbPool(sourceID string, m *Market, r *LegResult) {
	id := strings.ToLower(sourceID)
	for _, c := range m.Constituents {
		if p, ok := c.Pools[id]; ok {
			cp := *p
			cp.SqrtPriceX96, cp.Tick, cp.Liquidity = r.SqrtPriceX96, r.Tick, r.Liquidity
			w.pools[id] = &cp
			return
		}
	}
}

// Overlay replaces m's state with the world's wherever the world has it, and reports every field in
// which the two differed (m normally comes from an independent source of the same chain state, so a
// difference is a simulator error). Returns the number of pools carried over.
func (w *World) Overlay(m *Market) (int, []string) {
	var diffs []string
	n := 0
	if m.Mq != nil {
		if p, ok := w.pools[strings.ToLower(m.Mq.ConvPool)]; ok {
			// MqMarket can be shared with a clone; replace it instead of mutating it in place.
			mq := *m.Mq
			mq.Conv = p
			m.Mq = &mq
			n++
		}
	}
	for i := range m.Constituents {
		for id, p := range m.Constituents[i].Pools {
			q, ok := w.pools[strings.ToLower(id)]
			if !ok {
				continue
			}
			if p.SqrtPriceX96.Cmp(q.SqrtPriceX96) != 0 || p.Tick != q.Tick || p.Liquidity.Cmp(q.Liquidity) != 0 {
				diffs = append(diffs, fmt.Sprintf("pool %s: have %s/%d/%s, simulated %s/%d/%s", id, p.SqrtPriceX96,
					p.Tick, p.Liquidity, q.SqrtPriceX96, q.Tick, q.Liquidity))
			}
			m.Constituents[i].Pools[id] = q
			n++
		}
	}
	if v, ok := w.venues[strings.ToLower(m.Lot)]; ok {
		if m.Venue.SqrtPriceX96.Cmp(v.sqrtP) != 0 || m.Venue.Tick != v.tick {
			diffs = append(diffs, fmt.Sprintf("venue slot0: have %s/%d, simulated %s/%d", m.Venue.SqrtPriceX96,
				m.Venue.Tick, v.sqrtP, v.tick))
		}
		if !sameStanding(m.Venue.Standing, v.standing) {
			diffs = append(diffs, fmt.Sprintf("standing: have %+v, simulated %+v", m.Venue.Standing, v.standing))
		}
		if m.PmLotBalance.Cmp(v.pmLot) != 0 {
			diffs = append(diffs, fmt.Sprintf("PM LOT: have %s, simulated %s", m.PmLotBalance, v.pmLot))
		}
		m.Venue.SqrtPriceX96, m.Venue.Tick, m.Venue.Standing, m.PmLotBalance = v.sqrtP, v.tick, v.standing, v.pmLot
	}
	return n, diffs
}

func sameStanding(a, b *StandingPosition) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Liquidity.Cmp(b.Liquidity) == 0 && a.Lower == b.Lower && a.Upper == b.Upper
}

// PlainSwap is an ordinary third-party exact-input swap on one of m's constituent pools (stock's
// source sourceID), e.g. another route's hop before or after the basket swap.
func PlainSwap(cfg *Config, m *Market, stock, sourceID string, zeroForOne bool, amountIn,
	sqrtPriceLimitX96 *big.Int) (*LegResult, error) {
	id := strings.ToLower(sourceID)
	for _, c := range m.Constituents {
		if !strings.EqualFold(c.Stock, stock) {
			continue
		}
		p, ok := c.Pools[id]
		if !ok {
			break
		}
		fee, err := p.swapFee(cfg, id, zeroForOne)
		if err != nil {
			return nil, err
		}
		r := swapOnPool(fee, p, zeroForOne, false, amountIn, sqrtPriceLimitX96)
		return r, r.Err
	}
	return nil, fmt.Errorf("lotflow: no pool %s for %s", sourceID, stock)
}
