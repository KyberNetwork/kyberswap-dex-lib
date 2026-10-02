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
	sortedOraclesDenom    = big256.New(sortedOraclesDenominatorStr)
	oracleAdapterScaleDiv = big256.New(oracleAdapterScaleDownStr)
)

// convertWithRate ports FPMM._convertWithRate:
//
//	(amount * numerator * toDecimals) / (denominator * fromDecimals)
//
// Solidity 0.8 checked arithmetic reverts the moment any intermediate product
// overflows, so each multiplication is overflow-checked in the same order.
func convertWithRate(amount, fromDecimals, toDecimals, numerator, denominator *uint256.Int) (*uint256.Int, error) {
	var num, den uint256.Int
	if _, overflow := num.MulOverflow(amount, numerator); overflow {
		return nil, ErrOverflow
	}
	if _, overflow := num.MulOverflow(&num, toDecimals); overflow {
		return nil, ErrOverflow
	}
	if _, overflow := den.MulOverflow(denominator, fromDecimals); overflow {
		return nil, ErrOverflow
	}
	if den.IsZero() {
		return nil, ErrInvalidRate
	}
	return new(uint256.Int).Div(&num, &den), nil
}

// convertWithRateAndFee ports FPMM._convertWithRateAndFee:
//
//	(amount * numerator * toDecimals * feeNum) / (denominator * fromDecimals * feeDen)
func convertWithRateAndFee(amount, fromDecimals, toDecimals, numerator, denominator, feeNum,
	feeDen *uint256.Int) (*uint256.Int, error) {
	var num, den uint256.Int
	if _, overflow := num.MulOverflow(amount, numerator); overflow {
		return nil, ErrOverflow
	}
	if _, overflow := num.MulOverflow(&num, toDecimals); overflow {
		return nil, ErrOverflow
	}
	if _, overflow := num.MulOverflow(&num, feeNum); overflow {
		return nil, ErrOverflow
	}
	if _, overflow := den.MulOverflow(denominator, fromDecimals); overflow {
		return nil, ErrOverflow
	}
	if _, overflow := den.MulOverflow(&den, feeDen); overflow {
		return nil, ErrOverflow
	}
	if den.IsZero() {
		return nil, ErrInvalidRate
	}
	return new(uint256.Int).Div(&num, &den), nil
}

// scaleValue ports TradingLimitsV2.scaleValue: token units -> 15 decimals.
func scaleValue(value *uint256.Int, decimals uint8) (*uint256.Int, error) {
	if value.IsZero() {
		return new(uint256.Int), nil
	}
	var scaled uint256.Int
	if _, overflow := scaled.MulOverflow(value, limitInternalScale); overflow {
		return nil, ErrOverflow
	}
	return scaled.Div(&scaled, big256.TenPow(decimals)), nil
}

// applyTradingLimits ports TradingLimitsV2.applyTradingLimits + update +
// verify for one token. amountIn is the gross input (before the protocol fee
// leaves the pool), amountOut the output; one of them is zero for a
// single-direction swap. feeBps is lpFee + protocolFee. now is the timestamp
// the window resets are evaluated against. The returned TradingLimit shares
// no pointers with the input, so callers may keep both.
func applyTradingLimits(tl TradingLimit, amountIn, amountOut *uint256.Int, feeBps uint64,
	now uint64) (TradingLimit, error) {
	if tl.Limit0.Sign() == 0 && tl.Limit1.Sign() == 0 {
		return tl, nil
	}

	scaledIn, err := scaleValue(amountIn, tl.Decimals)
	if err != nil {
		return tl, err
	}
	scaledOut, err := scaleValue(amountOut, tl.Decimals)
	if err != nil {
		return tl, err
	}

	// scaledIn -= scaledIn * feeBps / BASIS_POINTS_DENOMINATOR
	var fee uint256.Int
	if _, overflow := fee.MulOverflow(scaledIn, uint256.NewInt(feeBps)); overflow {
		return tl, ErrOverflow
	}
	fee.Div(&fee, uBps)
	scaledIn.Sub(scaledIn, &fee)

	deltaFlow := new(int256.Int).Sub(big256.SInt256(scaledIn), big256.SInt256(scaledOut))
	if deltaFlow.Gt(maxInt96) || deltaFlow.Lt(minInt96) {
		return tl, ErrInt96Bounds
	}

	next := tl
	if deltaFlow.Sign() != 0 {
		if tl.Limit0.Sign() > 0 {
			if now > uint64(tl.LastUpdated0)+limitWindow0 {
				next.Netflow0 = new(int256.Int)
				next.LastUpdated0 = uint32(now)
			}
			if next.Netflow0, err = safeAddInt96(next.Netflow0, deltaFlow); err != nil {
				return tl, err
			}
		}
		if tl.Limit1.Sign() > 0 {
			if now > uint64(tl.LastUpdated1)+limitWindow1 {
				next.Netflow1 = new(int256.Int)
				next.LastUpdated1 = uint32(now)
			}
			if next.Netflow1, err = safeAddInt96(next.Netflow1, deltaFlow); err != nil {
				return tl, err
			}
		}
	}

	// verify
	if next.Limit0.Sign() > 0 {
		if next.Netflow0.Lt(new(int256.Int).Neg(next.Limit0)) || next.Netflow0.Gt(next.Limit0) {
			return tl, ErrL0LimitExceeded
		}
	}
	if next.Limit1.Sign() > 0 {
		if next.Netflow1.Lt(new(int256.Int).Neg(next.Limit1)) || next.Netflow1.Gt(next.Limit1) {
			return tl, ErrL1LimitExceeded
		}
	}
	return next, nil
}

func safeAddInt96(a, b *int256.Int) (*int256.Int, error) {
	c := new(int256.Int).Add(a, b)
	if c.Lt(minInt96) || c.Gt(maxInt96) {
		return nil, ErrInt96Bounds
	}
	return c, nil
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
	return !(month == 1 && day == 1)
}
