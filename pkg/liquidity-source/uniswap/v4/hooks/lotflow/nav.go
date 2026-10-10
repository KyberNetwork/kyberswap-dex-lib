package lotflow

import (
	"errors"
	"math/big"
)

// NAV of one LOT, as the hook reads it from NavGuard.checkedNavPerUnit18 (MemeFiLP
// contracts/src/NavGuard.sol) with spot prices from BasketFactory.spotUsd18 / SpotLib.price18.

// NavGuardParams is NavGuard.Params.
type NavGuardParams struct {
	NavTolBps      uint64
	StaleTolBps    uint64
	MaxFeedAge     uint64 // seconds
	MinCoverageBps uint64
}

// FeedState is a constituent's Chainlink feed: latestRoundData().answer / updatedAt and decimals().
type FeedState struct {
	Answer    *big.Int
	Decimals  int
	UpdatedAt uint64
}

// NavConstituent is one basket constituent as NavGuard sees it.
type NavConstituent struct {
	Unit          *big.Int // wei of stock per 1e18 LOT
	SpotUsd18     *big.Int // BasketFactory.spotUsd18(stock)
	StockDecimals int
	Feed          *FeedState // nil = NavGuard.feedOf(stock) == address(0)
	OraclePaused  bool       // stock.oraclePaused() == true (an unreadable call counts as false)
}

// SpotUsd18 is SpotLib.price18: USD (1e18) per whole stock token from the registry price pool's
// sqrtPriceX96.
func SpotUsd18(sqrtPriceX96 *big.Int, stockIsCurrency0 bool, stockDec, stableDec int) (*big.Int, error) {
	if sqrtPriceX96.Sign() == 0 {
		return nil, errors.New("lotflow: SpotLib ZeroSqrtPrice")
	}
	ratioX128 := mulDiv(sqrtPriceX96, sqrtPriceX96, new(big.Int).Lsh(big.NewInt(1), 64))
	if stockIsCurrency0 {
		return mulDiv(ratioX128, new(big.Int).Mul(bigE18, pow10(stockDec)),
			new(big.Int).Mul(bigQ128, pow10(stableDec))), nil
	}
	if ratioX128.Sign() == 0 {
		return nil, errors.New("lotflow: SpotLib division by zero")
	}
	p := mulDiv(new(big.Int).Mul(bigQ128, bigE18), pow10(stockDec), ratioX128)
	return p.Quo(p, pow10(stableDec)), nil
}

// CheckedNavPerUnit18 is NavGuard.checkedNavPerUnit18: the spot NAV of 1e18 LOT, required to agree
// with the Chainlink NAV (spot standing in where a stock has no feed) within navTolBps, or within
// staleTolBps when any feed is stale (updatedAt + maxFeedAge < timestamp) or any fed stock reports
// oraclePaused; and the fed share of spot NAV must reach minCoverageBps. timestamp is block.timestamp.
func CheckedNavPerUnit18(cs []NavConstituent, p NavGuardParams, timestamp uint64) (*big.Int, error) {
	spot, cl, covered := new(big.Int), new(big.Int), new(big.Int)
	stale := false
	for _, c := range cs {
		one := pow10(c.StockDecimals)
		spotVal := mulDiv(c.Unit, c.SpotUsd18, one)
		spot.Add(spot, spotVal)
		if c.Feed == nil {
			cl.Add(cl, spotVal)
			continue
		}
		if c.Feed.Answer.Sign() <= 0 || c.Feed.UpdatedAt == 0 {
			return nil, ErrBadFeedAnswer
		}
		feedStale := c.Feed.UpdatedAt+p.MaxFeedAge < timestamp || c.OraclePaused
		px18 := new(big.Int).Mul(c.Feed.Answer, pow10(18-c.Feed.Decimals))
		cl.Add(cl, mulDiv(c.Unit, px18, one))
		covered.Add(covered, spotVal)
		if feedStale {
			stale = true
		}
	}
	if spot.Sign() == 0 {
		return nil, ErrZeroNav
	}
	coveredBps := new(big.Int).Quo(new(big.Int).Mul(covered, bigBps), spot).Uint64()
	if coveredBps < p.MinCoverageBps {
		return nil, &FeedCoverageTooLowError{CoveredBps: coveredBps, MinCoverageBps: p.MinCoverageBps}
	}
	diff := new(big.Int).Sub(spot, cl)
	diff.Abs(diff)
	tol := p.NavTolBps
	if stale {
		tol = p.StaleTolBps
	}
	if new(big.Int).Mul(diff, bigBps).Cmp(new(big.Int).Mul(cl, new(big.Int).SetUint64(tol))) > 0 {
		return nil, &NavMismatchError{Spot: spot, Chainlink: cl}
	}
	return spot, nil
}
