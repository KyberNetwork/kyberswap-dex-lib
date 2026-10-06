package flywheelfun

import (
	"encoding/json"
	"errors"

	"github.com/holiman/uint256"
)

var (
	ErrMath     = errors.New("flywheel: uint256 overflow or invalid denominator")
	ErrAmount   = errors.New("flywheel: invalid or dust amount")
	ErrProtocol = errors.New("flywheel: protocol fee above cap")
	ErrCurve    = errors.New("flywheel: insufficient curve liquidity")
	ErrPartial  = errors.New("flywheel: unexpected partial market fill")
)

// Split mirrors the deployed NativeFeeMath allocation, in asset base units.
type Split struct{ Net, Rewards, Burn, Liquidity, Platform uint256.Int }

// mulDiv preserves Solidity's 512-bit intermediate and checked uint256 result.
func mulDiv(x, y, d *uint256.Int, up bool) (uint256.Int, error) {
	var q, r uint256.Int
	if d.IsZero() {
		return q, ErrMath
	}
	if _, overflow := q.MulDivOverflow(x, y, d); overflow {
		return q, ErrMath
	}
	if up {
		r.MulMod(x, y, d)
		if !r.IsZero() {
			if _, overflow := q.AddOverflow(&q, uint256.NewInt(1)); overflow {
				return q, ErrMath
			}
		}
	}
	return q, nil
}
func ratio(x *uint256.Int, n, d uint64, up bool) (uint256.Int, error) {
	return mulDiv(x, uint256.NewInt(n), uint256.NewInt(d), up)
}

func BuyPlatform(gross *uint256.Int, protocol uint32) (uint256.Int, error) {
	if protocol > 1000 {
		return uint256.Int{}, ErrProtocol
	}
	return ratio(gross, 9500*1_000_000-969500*uint64(protocol), 1_000_000*(1_000_000-uint64(protocol)), true)
}
func BuyQuote(quote *uint256.Int, protocol uint32) (s Split, err error) {
	if protocol > 1000 {
		return s, ErrProtocol
	}
	factor := 1_000_000 - uint64(protocol)
	denom := 990500*1_000_000 - 30500*uint64(protocol)
	fee, err := ratio(quote, 30500*factor, denom, true)
	if err != nil {
		return s, err
	}
	s.Net.Sub(quote, &fee)
	s.Rewards, err = ratio(quote, 29500*factor, denom, false)
	if err != nil {
		return s, err
	}
	s.Burn, err = ratio(quote, 500*factor, denom, false)
	if err != nil {
		return s, err
	}
	s.Liquidity.Sub(&fee, &s.Rewards)
	s.Liquidity.Sub(&s.Liquidity, &s.Burn)
	return s, nil
}
func SellQuote(quote *uint256.Int, protocol uint32) (s Split, err error) {
	if protocol > 1000 {
		return s, ErrProtocol
	}
	s.Burn, err = ratio(quote, 500, 1_000_000-uint64(protocol), false)
	if err != nil {
		return s, err
	}
	s.Liquidity = s.Burn
	s.Net.Sub(quote, &s.Burn)
	s.Net.Sub(&s.Net, &s.Liquidity)
	return s, nil
}
func SellETH(realized, grossQuote *uint256.Int, protocol uint32) (s Split, err error) {
	reserves, err := SellQuote(grossQuote, protocol)
	if err != nil {
		return s, err
	}
	if reserves.Net.IsZero() {
		return s, ErrAmount
	}
	value, err := mulDiv(realized, grossQuote, &reserves.Net, false)
	if err != nil {
		return s, err
	}
	d := 1_000_000 - uint64(protocol)
	s.Net, err = ratio(&value, 960000, d, false)
	if err != nil {
		return s, err
	}
	s.Rewards, err = ratio(&value, 29500, d, false)
	if err != nil {
		return s, err
	}
	if _, overflow := s.Platform.SubOverflow(realized, &s.Net); overflow {
		return s, ErrMath
	}
	if _, overflow := s.Platform.SubOverflow(&s.Platform, &s.Rewards); overflow {
		return s, ErrMath
	}
	return s, nil
}

type CurveState struct {
	QuoteReserve uint256.Int `json:"quoteReserve"`
	TokenReserve uint256.Int `json:"tokenReserve"`
	VirtualQuote uint256.Int `json:"virtualQuote"`
	Threshold    uint256.Int `json:"threshold"`
	Graduated    bool        `json:"graduated"`
}

// Value fields avoid shared mutable integers, while explicit decimal JSON keeps
// the persisted format independent of uint256's pointer marshaler.
type curveJSON struct {
	QuoteReserve string `json:"quoteReserve"`
	TokenReserve string `json:"tokenReserve"`
	VirtualQuote string `json:"virtualQuote"`
	Threshold    string `json:"threshold"`
	Graduated    bool   `json:"graduated"`
}

func (c CurveState) MarshalJSON() ([]byte, error) {
	return json.Marshal(curveJSON{c.QuoteReserve.Dec(), c.TokenReserve.Dec(), c.VirtualQuote.Dec(), c.Threshold.Dec(), c.Graduated})
}
func (c *CurveState) UnmarshalJSON(data []byte) error {
	var wire curveJSON
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	var next CurveState
	next.Graduated = wire.Graduated
	for _, p := range []struct {
		s string
		d *uint256.Int
	}{{wire.QuoteReserve, &next.QuoteReserve}, {wire.TokenReserve, &next.TokenReserve}, {wire.VirtualQuote, &next.VirtualQuote}, {wire.Threshold, &next.Threshold}} {
		x, err := uint256.FromDecimal(p.s)
		if err != nil {
			return err
		}
		p.d.Set(x)
	}
	*c = next
	return nil
}

// CurveTrade is the full-fill curve helper. The settlement simulator uses
// CurveFill below to also model graduation and its returned quote amount.
func CurveTrade(c CurveState, amount *uint256.Int, buy bool) (out uint256.Int, next CurveState, err error) {
	return curveTrade(c, amount, buy, false)
}
func CurveFill(c CurveState, amount *uint256.Int, buy bool) (out uint256.Int, next CurveState, used uint256.Int, err error) {
	used = *amount
	if buy {
		var backing, remaining uint256.Int
		if _, bad := backing.SubOverflow(&c.QuoteReserve, &c.VirtualQuote); bad {
			return out, c, used, ErrCurve
		}
		if _, bad := remaining.SubOverflow(&c.Threshold, &backing); bad || remaining.IsZero() {
			return out, c, used, ErrCurve
		}
		if used.Gt(&remaining) {
			used = remaining
		}
	}
	out, next, err = curveTrade(c, &used, buy, true)
	return
}
func curveTrade(c CurveState, amount *uint256.Int, buy bool, allowGraduation bool) (out uint256.Int, next CurveState, err error) {
	next = c
	if amount.IsZero() || c.QuoteReserve.IsZero() || c.TokenReserve.IsZero() || c.Graduated {
		return out, next, ErrCurve
	}
	var real, remaining uint256.Int
	if _, overflow := real.SubOverflow(&c.QuoteReserve, &c.VirtualQuote); overflow {
		return out, next, ErrCurve
	}
	if _, overflow := remaining.SubOverflow(&c.Threshold, &real); overflow {
		return out, next, ErrCurve
	}
	if buy {
		if amount.Cmp(&remaining) >= 0 && !allowGraduation {
			return out, next, ErrPartial
		}
		if _, overflow := next.QuoteReserve.AddOverflow(&c.QuoteReserve, amount); overflow {
			return out, c, ErrMath
		}
		next.TokenReserve, err = mulDiv(&c.QuoteReserve, &c.TokenReserve, &next.QuoteReserve, true)
		if err != nil {
			return out, c, err
		}
		out.Sub(&c.TokenReserve, &next.TokenReserve)
		next.Graduated = amount.Eq(&remaining)
	} else {
		if _, overflow := next.TokenReserve.AddOverflow(&c.TokenReserve, amount); overflow {
			return out, c, ErrMath
		}
		next.QuoteReserve, err = mulDiv(&c.QuoteReserve, &c.TokenReserve, &next.TokenReserve, true)
		if err != nil {
			return out, c, err
		}
		if next.QuoteReserve.Cmp(&c.VirtualQuote) < 0 {
			return out, c, ErrCurve
		}
		out.Sub(&c.QuoteReserve, &next.QuoteReserve)
	}
	if out.IsZero() {
		return out, c, ErrAmount
	}
	return out, next, nil
}
