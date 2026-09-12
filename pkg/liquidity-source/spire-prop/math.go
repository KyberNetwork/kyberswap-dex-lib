package spireprop

import (
	"github.com/holiman/uint256"
)

var wad = uint256.NewInt(1_000_000_000_000_000_000)
var bps = uint256.NewInt(10_000)

func mul(a, b *uint256.Int) (uint256.Int, error) {
	var z uint256.Int
	if _, overflow := z.MulOverflow(a, b); overflow {
		return z, ErrOverflow
	}
	return z, nil
}

func add(a, b *uint256.Int) (uint256.Int, error) {
	var z uint256.Int
	if _, overflow := z.AddOverflow(a, b); overflow {
		return z, ErrOverflow
	}
	return z, nil
}

// Solidity's ceilDiv(a,b), including the no-overflow operation order.
func ceilDiv(a, b *uint256.Int) uint256.Int {
	if a.IsZero() {
		return uint256.Int{}
	}
	var z uint256.Int
	z.SubUint64(a, 1).Div(&z, b).AddUint64(&z, 1)
	return z
}

func (s *PoolSimulator) sidePrice(side *Side) uint256.Int {
	var scale, price uint256.Int
	scale.SetUint64(uint64(10_000 + int32(side.SpreadBps)))
	// Mid is uint80 and scale <= 42767; validated state cannot overflow.
	price.Mul(&s.Extra.Mid, &scale).Div(&price, bps)
	return price
}

func (s *PoolSimulator) atomic(knot *Knot) (q, c uint256.Int, err error) {
	q, err = mul(&knot.Q, &s.Extra.QUnit)
	if err != nil {
		return
	}
	c, err = mul(&knot.Extra, &s.Extra.CUnit)
	return
}

func (s *PoolSimulator) effectiveMax(side *Side) (uint256.Int, error) {
	if len(side.Knots) == 0 || side.DepthBps == 0 {
		return uint256.Int{}, nil
	}
	q, _, err := s.atomic(&side.Knots[len(side.Knots)-1])
	if err != nil {
		return q, err
	}
	var scale uint256.Int
	scale.SetUint64(uint64(side.DepthBps))
	q, err = mul(&q, &scale)
	q.Div(&q, bps)
	return q, err
}

func interp(prevQ, prevC, qA, cA, x *uint256.Int) (uint256.Int, error) {
	var dc, dx, dq uint256.Int
	dc.Sub(cA, prevC)
	dx.Sub(x, prevQ)
	dq.Sub(qA, prevQ)
	product, err := mul(&dc, &dx)
	if err != nil {
		return uint256.Int{}, err
	}
	extra := ceilDiv(&product, &dq)
	return add(prevC, &extra)
}

func (s *PoolSimulator) extraAt(side *Side, x *uint256.Int) (uint256.Int, error) {
	var prevQ, prevC uint256.Int
	for i := range side.Knots {
		qA, cA, err := s.atomic(&side.Knots[i])
		if err != nil {
			return uint256.Int{}, err
		}
		if x.Cmp(&qA) <= 0 {
			return interp(&prevQ, &prevC, &qA, &cA, x)
		}
		prevQ, prevC = qA, cA
	}
	return prevC, nil
}

func (s *PoolSimulator) sell(amount *uint256.Int) (out, cursor uint256.Int, err error) {
	side := &s.Extra.Bid
	cursor, err = add(&side.Filled, amount)
	if err != nil {
		// BaibaiCurveBook._sellBase uses Math.tryAdd and maps a failed add
		// to Fail.BeyondDepth. Preserve that result; the multiplication below
		// instead uses Solidity checked arithmetic and remains ErrOverflow.
		return out, cursor, ErrDepth
	}
	maxQ, err := s.effectiveMax(side)
	if err != nil {
		return out, cursor, err
	}
	if cursor.Gt(&maxQ) {
		return out, cursor, ErrDepth
	}
	price := s.sidePrice(side)
	mid, err := mul(&price, amount)
	if err != nil {
		return out, cursor, err
	}
	mid.Div(&mid, wad)
	end, err := s.extraAt(side, &cursor)
	if err != nil {
		return out, cursor, err
	}
	start, err := s.extraAt(side, &side.Filled)
	if err != nil {
		return out, cursor, err
	}
	end.Sub(&end, &start)
	if end.Cmp(&mid) >= 0 {
		return out, cursor, ErrDepth
	}
	out.Sub(&mid, &end)
	return
}

func (s *PoolSimulator) buy(amount *uint256.Int) (out, cursor uint256.Int, err error) {
	side := &s.Extra.Ask
	cursor = side.Filled
	maxQ, err := s.effectiveMax(side)
	if err != nil {
		return out, cursor, err
	}
	if cursor.Cmp(&maxQ) >= 0 {
		return out, cursor, ErrDepth
	}
	price, remaining := s.sidePrice(side), *amount
	var prevQ, prevC uint256.Int
	for i := range side.Knots {
		if remaining.IsZero() {
			break
		}
		qA, cA, e := s.atomic(&side.Knots[i])
		if e != nil {
			return out, cursor, e
		}
		if qA.Gt(&cursor) {
			endQ := qA
			if maxQ.Lt(&endQ) {
				endQ = maxQ
			}
			var delta uint256.Int
			delta.Sub(&endQ, &cursor)
			product, e := mul(&price, &delta)
			if e != nil {
				return out, cursor, e
			}
			cost := ceilDiv(&product, wad)
			endExtra, e := interp(&prevQ, &prevC, &qA, &cA, &endQ)
			if e != nil {
				return out, cursor, e
			}
			startExtra, e := interp(&prevQ, &prevC, &qA, &cA, &cursor)
			if e != nil {
				return out, cursor, e
			}
			// Preserve Solidity's left-to-right checked addition before subtracting.
			cost, e = add(&cost, &endExtra)
			if e != nil {
				return out, cursor, e
			}
			cost.Sub(&cost, &startExtra)
			if remaining.Lt(&cost) {
				var dq uint256.Int
				if _, overflow := dq.MulDivOverflow(&remaining, &delta, &cost); overflow {
					return out, cursor, ErrOverflow
				}
				if dq.IsZero() {
					return out, cursor, ErrDepth
				}
				cursor.Add(&cursor, &dq)
				remaining.Clear()
			} else {
				remaining.Sub(&remaining, &cost)
				cursor = endQ
			}
			if cursor.Eq(&maxQ) {
				break
			}
		}
		prevQ, prevC = qA, cA
	}
	if !remaining.IsZero() {
		return out, cursor, ErrDepth
	}
	out.Sub(&cursor, &side.Filled)
	if out.IsZero() {
		return out, cursor, ErrDepth
	}
	return
}
