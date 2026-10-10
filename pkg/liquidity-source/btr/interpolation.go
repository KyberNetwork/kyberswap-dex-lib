package btr

import "math/big"

// interpolate returns the output for amountIn from a sampled ladder
// (ascending and monotonic in In). Between samples it is piecewise linear;
// below the first sample it scales linearly from the origin; above the last
// sample it fails closed rather than extrapolating a curve it has not probed.
//
// The ladder is a cache of the on-chain `getSwapQuote` view. It is never an
// execution guarantee (see Extra); its job is to let the route finder rank a
// BTR leg without an eth_call per candidate amount.
func interpolate(points []Point, amountIn *big.Int) (*big.Int, error) {
	if amountIn == nil || amountIn.Sign() <= 0 {
		return nil, ErrBTRInvalidAmount
	}
	if len(points) == 0 {
		return nil, ErrBTRNoRoute
	}

	firstIn, ok := new(big.Int).SetString(points[0].In, 10)
	if !ok {
		return nil, ErrBTRBadState
	}
	firstOut, ok := new(big.Int).SetString(points[0].Out, 10)
	if !ok {
		return nil, ErrBTRBadState
	}
	if amountIn.Cmp(firstIn) <= 0 {
		if firstIn.Sign() == 0 {
			return nil, ErrBTRNoFill
		}
		out := new(big.Int).Mul(firstOut, amountIn)
		return out.Div(out, firstIn), nil
	}

	last := points[len(points)-1]
	lastIn, ok := new(big.Int).SetString(last.In, 10)
	if !ok {
		return nil, ErrBTRBadState
	}
	if amountIn.Cmp(lastIn) > 0 {
		return nil, ErrBTRAmountTooLarge
	}

	for i := 1; i < len(points); i++ {
		x1, ok := new(big.Int).SetString(points[i].In, 10)
		if !ok {
			return nil, ErrBTRBadState
		}
		if amountIn.Cmp(x1) > 0 {
			continue
		}
		x0, ok := new(big.Int).SetString(points[i-1].In, 10)
		if !ok {
			return nil, ErrBTRBadState
		}
		y0, ok := new(big.Int).SetString(points[i-1].Out, 10)
		if !ok {
			return nil, ErrBTRBadState
		}
		y1, ok := new(big.Int).SetString(points[i].Out, 10)
		if !ok {
			return nil, ErrBTRBadState
		}
		dx := new(big.Int).Sub(x1, x0)
		if dx.Sign() <= 0 {
			return nil, ErrBTRBadState
		}
		// out = y0 + (y1 - y0) * (amountIn - x0) / dx
		out := new(big.Int).Sub(y1, y0)
		out.Mul(out, new(big.Int).Sub(amountIn, x0))
		out.Div(out, dx)
		return out.Add(out, y0), nil
	}
	return nil, ErrBTRAmountTooLarge
}
