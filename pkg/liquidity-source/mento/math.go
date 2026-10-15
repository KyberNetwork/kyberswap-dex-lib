package mento

import (
	"time"

	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

var (
	uBps                  = uint256.NewInt(bps)
	limitInternalScale    = big256.TenPow(limitInternalDecimals)
	maxInt96              = int256.MustFromDec("39614081257132168796771975167")  // 2^95 - 1
	minInt96              = int256.MustFromDec("-39614081257132168796771975168") // -2^95
	sortedOraclesDenom    = big256.TenPow(24)                                    // SortedOracles fixidity denominator
	oracleAdapterScaleDiv = big256.TenPow(6)                                     // OracleAdapter divides both rate terms by 1e6
)

// quoteRate holds the constant factors of FPMM._convertWithRate(AndFee) for one swap
// direction, pre-multiplied so a quote is one checked Mul and one Div:
//
//	out   = amount * num * toDec * feeNum / (den * fromDec * BASIS_POINTS_DENOMINATOR)
//	gross = amount * num * toDec / (den * fromDec)
type quoteRate struct {
	numTo    uint256.Int // num * toDec
	outNum   uint256.Int // num * toDec * feeNum
	grossDen uint256.Int // den * fromDec
	outDen   uint256.Int // den * fromDec * BASIS_POINTS_DENOMINATOR
}

// newQuoteRate errors when a constant factor overflows: FPMM would revert on every amount.
func newQuoteRate(num, den, fromDec, toDec *uint256.Int, feeNum uint64) (quoteRate, error) {
	var r quoteRate
	if _, overflow := r.numTo.MulOverflow(num, toDec); overflow {
		return r, ErrOverflow
	}
	if _, overflow := r.outNum.MulOverflow(&r.numTo, uint256.NewInt(feeNum)); overflow {
		return r, ErrOverflow
	}
	if _, overflow := r.grossDen.MulOverflow(den, fromDec); overflow {
		return r, ErrOverflow
	}
	if _, overflow := r.outDen.MulOverflow(&r.grossDen, uBps); overflow {
		return r, ErrOverflow
	}
	return r, nil
}

// amountOut writes FPMM.getAmountOut into out and gross - out into fee. The product is
// overflow-checked, not 512-bit: Solidity reverts once amount * num * ... overflows.
func (r *quoteRate) amountOut(amountIn, out, fee *uint256.Int) error {
	if r.numTo.IsZero() || r.grossDen.IsZero() {
		return ErrInvalidRate
	}
	if _, overflow := out.MulOverflow(amountIn, &r.outNum); overflow {
		return ErrOverflow
	}
	out.Div(out, &r.outDen)
	// amountIn * numTo <= amountIn * outNum while feeNum >= 1, so it cannot overflow.
	fee.Mul(amountIn, &r.numTo)
	fee.Div(fee, &r.grossDen)
	if fee.Lt(out) {
		fee.Clear()
	} else {
		fee.Sub(fee, out)
	}
	return nil
}

// scaleValue ports TradingLimitsV2.scaleValue into z: token units -> 15 decimals.
func scaleValue(z, value *uint256.Int, decimals uint8) error {
	if _, overflow := z.MulOverflow(value, limitInternalScale); overflow {
		return ErrOverflow
	}
	z.Div(z, big256.TenPow(decimals))
	return nil
}

// applyTradingLimits ports TradingLimitsV2.applyTradingLimits + update + verify for one
// token. amountIn is the gross input, amountOut the output (one is zero). Limit pointers
// are shared with tl and never written in place; changed netflows get fresh pointers.
func applyTradingLimits(tl TradingLimit, amountIn, amountOut *uint256.Int, feeBps uint64,
	now uint64) (TradingLimit, error) {
	if tl.Limit0.Sign() == 0 && tl.Limit1.Sign() == 0 {
		return tl, nil
	}

	var in, out, fee uint256.Int
	if err := scaleValue(&in, amountIn, tl.Decimals); err != nil {
		return tl, err
	}
	if err := scaleValue(&out, amountOut, tl.Decimals); err != nil {
		return tl, err
	}
	// scaledIn -= scaledIn * feeBps / BASIS_POINTS_DENOMINATOR
	if _, overflow := fee.MulOverflow(&in, fee.SetUint64(feeBps)); overflow {
		return tl, ErrOverflow
	}
	in.Sub(&in, fee.Div(&fee, uBps))

	var deltaFlow int256.Int
	(*uint256.Int)(&deltaFlow).Sub(&in, &out)
	if deltaFlow.Gt(maxInt96) || deltaFlow.Lt(minInt96) {
		return tl, ErrInt96Bounds
	}

	next := tl
	if deltaFlow.Sign() != 0 {
		netflows := new([2]int256.Int)
		if tl.Limit0.Sign() > 0 {
			if now <= uint64(tl.LastUpdated0)+limitWindow0 {
				netflows[0].Set(tl.Netflow0)
			} else {
				next.LastUpdated0 = uint32(now)
			}
			if err := safeAddInt96(&netflows[0], &deltaFlow); err != nil {
				return tl, err
			}
			next.Netflow0 = &netflows[0]
		}
		if tl.Limit1.Sign() > 0 {
			if now <= uint64(tl.LastUpdated1)+limitWindow1 {
				netflows[1].Set(tl.Netflow1)
			} else {
				next.LastUpdated1 = uint32(now)
			}
			if err := safeAddInt96(&netflows[1], &deltaFlow); err != nil {
				return tl, err
			}
			next.Netflow1 = &netflows[1]
		}
	}

	// verify
	var neg int256.Int
	if next.Limit0.Sign() > 0 && (next.Netflow0.Lt(neg.Neg(next.Limit0)) || next.Netflow0.Gt(next.Limit0)) {
		return tl, ErrL0LimitExceeded
	}
	if next.Limit1.Sign() > 0 && (next.Netflow1.Lt(neg.Neg(next.Limit1)) || next.Netflow1.Gt(next.Limit1)) {
		return tl, ErrL1LimitExceeded
	}
	return next, nil
}

func safeAddInt96(z, delta *int256.Int) error {
	if z.Add(z, delta); z.Lt(minInt96) || z.Gt(maxInt96) {
		return ErrInt96Bounds
	}
	return nil
}

// isFXMarketOpen ports MarketHoursBreaker.isFXMarketOpen: closed from Friday
// 21:00 UTC until Sunday 23:00 UTC, on Dec 25 and Jan 1, and from 22:00 UTC
// on Dec 24 and Dec 31.
func isFXMarketOpen(timestamp uint64) bool {
	t := time.Unix(int64(timestamp), 0).UTC()
	dow := int(t.Weekday()) // Sunday = 0
	if dow == 0 {
		dow = 7 // BokkyPooBahsDateTimeLibrary: Monday = 1 ... Sunday = 7
	}
	hour := t.Hour()

	if (dow == 5 && hour >= 21) || dow == 6 || (dow == 7 && hour < 23) {
		return false
	}

	month, day := int(t.Month()), t.Day()
	if month == 12 {
		if day == 24 || day == 31 {
			return hour < 22
		}
		return day != 25
	}
	return month != 1 || day != 1
}
