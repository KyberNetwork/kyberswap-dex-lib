package wombat

import (
	"errors"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"

	dsmathInt "github.com/KyberNetwork/blockchain-toolkit/dsmath"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/dsmath"
)

var (
	ErrWombatCovRatioLimitExceeded = errors.New("wombat: cov ratio limit exceeded")
	ErrWombatCashNotEnough         = errors.New("wombat: cash not enough")
)

type PoolSimulator struct {
	startCovRatio *uint256.Int
	endCovRatio   *uint256.Int
	haircutRate   *uint256.Int
	ampFactor     *uint256.Int
}

// _highCovRatioFeePoolV2__highCovRatioFee
// https://github.com/wombat-exchange/v1-core/blob/d9bb07de2272e0d9cb4c640671a925896b64fc14/contracts/wombat-core/pool/HighCovRatioFeePoolV2.sol#L43
func (s *PoolSimulator) _highCovRatioFeePoolV2__highCovRatioFee(initCovRatio *uint256.Int, finalCovRatio *uint256.Int) (*uint256.Int, error) {
	if finalCovRatio.Gt(s.endCovRatio) {
		return nil, ErrWombatCovRatioLimitExceeded
	}

	if finalCovRatio.Cmp(s.startCovRatio) <= 0 || finalCovRatio.Cmp(initCovRatio) <= 0 {
		return number.Zero, nil
	}

	var a *uint256.Int
	if initCovRatio.Cmp(s.startCovRatio) <= 0 {
		a = number.Zero
	} else {
		a = new(uint256.Int).Mul(
			new(uint256.Int).Sub(initCovRatio, s.startCovRatio),
			new(uint256.Int).Sub(initCovRatio, s.startCovRatio),
		)
	}

	b := new(uint256.Int).Mul(
		new(uint256.Int).Sub(finalCovRatio, s.startCovRatio),
		new(uint256.Int).Sub(finalCovRatio, s.startCovRatio),
	)

	fee := dsmath.WDiv(
		new(uint256.Int).Div(new(uint256.Int).Div(
			new(uint256.Int).Sub(b, a),
			new(uint256.Int).Sub(finalCovRatio, initCovRatio),
		), number.Number_2),
		new(uint256.Int).Sub(s.endCovRatio, s.startCovRatio),
	)

	return fee, nil
}

// _highCovRatioFeePoolV2__findUpperBound
// https://github.com/wombat-exchange/v1-core/blob/d9bb07de2272e0d9cb4c640671a925896b64fc14/contracts/wombat-core/pool/HighCovRatioFeePoolV2.sol#L116
func (s *PoolSimulator) _highCovRatioFeePoolV2__findUpperBound(fromAsset, toAsset Asset, toAmount *uint256.Int) (*uint256.Int, error) {
	decimals := fromAsset.UnderlyingTokenDecimals
	toWadFactor := dsmath.ToWAD(number.Number_1, decimals)
	high := dsmath.FromWAD(
		new(uint256.Int).Sub(
			dsmath.WMul(fromAsset.Liability, s.endCovRatio),
			fromAsset.Cash,
		),
		decimals,
	)

	low := number.Number_1

	quote, _, err := s._highCovRatioFeePoolV2__quoteFrom(fromAsset, toAsset, new(uint256.Int).Mul(high, toWadFactor).ToBig())
	if err != nil {
		return nil, err
	}

	if quote.Lt(toAmount) {
		return nil, ErrWombatCovRatioLimitExceeded
	}

	for low.Lt(high) {
		mid := new(uint256.Int).Div(
			new(uint256.Int).Add(low, high),
			number.Number_2,
		)

		quote, _, err := s._highCovRatioFeePoolV2__quoteFrom(fromAsset, toAsset, new(uint256.Int).Mul(mid, toWadFactor).ToBig())
		if err != nil {
			return nil, err
		}

		if quote.Cmp(toAmount) >= 0 {
			high = mid
		} else {
			low = new(uint256.Int).Add(mid, number.Number_1)
		}
	}

	return new(uint256.Int).Mul(high, toWadFactor), nil
}

// _highCovRatioFeePoolV2__quoteFrom
func (s *PoolSimulator) _highCovRatioFeePoolV2__quoteFrom(fromAsset, toAsset Asset, fromAmount *big.Int) (*uint256.Int, *uint256.Int, error) {
	return nil, nil, nil
}

// _poolV2__quoteFrom
// https://github.com/wombat-exchange/v1-core/blob/d9bb07de2272e0d9cb4c640671a925896b64fc14/contracts/wombat-core/pool/PoolV2.sol#L761
func (s *PoolSimulator) _poolV2__quoteFrom(fromAsset, toAsset Asset, fromAmount *big.Int) (*uint256.Int, *uint256.Int, error) {
	if fromAmount.Cmp(integer.Zero()) < 0 {
		fromAmount = dsmathInt.WDiv(fromAmount, new(big.Int).Sub(dsmathInt.WAD, s.haircutRate.ToBig()))
	}

	fromCash, fromLiability := fromAsset.Cash, fromAsset.Liability
	toCash := toAsset.Cash

	scaleFactor := number.Number_1e18

	if !scaleFactor.Eq(dsmath.WAD) {
		fromCash = new(uint256.Int).Div(new(uint256.Int).Mul(fromCash, scaleFactor), number.Number_1e18)
		fromLiability = new(uint256.Int).Div(new(uint256.Int).Mul(fromLiability, scaleFactor), number.Number_1e18)
		fromAmount = new(big.Int).Div(new(big.Int).Mul(fromAmount, scaleFactor.ToBig()), integer.TenPow(18))
	}

	idealToAmount, err := CoreV2._swapQuoteFunc(
		fromCash.ToBig(),
		toCash.ToBig(),
		fromLiability.ToBig(),
		toAsset.Liability.ToBig(),
		fromAmount,
		s.ampFactor.ToBig(),
	)
	if err != nil {
		return nil, nil, err
	}

	if (fromAmount.Cmp(integer.Zero()) > 0 && toCash.Lt(idealToAmount)) ||
		(fromAmount.Cmp(integer.Zero()) < 0 && fromAsset.Cash.Lt(uint256.MustFromBig(new(big.Int).Neg(fromAmount)))) {
		return nil, nil, ErrWombatCashNotEnough
	}

	var (
		actualToAmount *uint256.Int
		haircut        *uint256.Int
	)
	if fromAmount.Cmp(integer.Zero()) > 0 {
		haircut = dsmath.WMul(idealToAmount, s.haircutRate)
		actualToAmount = new(uint256.Int).Sub(idealToAmount, haircut)
	} else {
		actualToAmount = idealToAmount
		haircut = dsmath.WMul(uint256.MustFromBig(new(big.Int).Neg(fromAmount)), s.haircutRate)
	}

	return actualToAmount, haircut, nil
}
