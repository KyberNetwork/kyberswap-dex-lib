package dexv2

import (
	"math/big"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func _calculateVars(dexVariables2, token0ExchangePricesAndConfig, token1ExchangePricesAndConfig *big.Int, snapshotTimestamp uint64) (CalculatedVars, error) {
	// NOTE: Should not mutate CalculatedVars's fields (readonly)
	var tmp big.Int
	tmp.Set(dexVariables2).
		Rsh(&tmp, BITS_DEX_V2_VARIABLES2_TOKEN_0_DECIMALS).
		And(&tmp, X4)

	decimals := tmp.Int64()
	if decimals == 15 {
		decimals = 18
	}

	token0NumeratorPrecision, token0DenominatorPrecision := _calculateNumeratorAndDenominatorPrecisions(decimals)

	tmp.Set(dexVariables2).
		Rsh(&tmp, BITS_DEX_V2_VARIABLES2_TOKEN_1_DECIMALS).
		And(&tmp, X4)

	decimals = tmp.Int64()
	if decimals == 15 {
		decimals = 18
	}
	token1NumeratorPrecision, token1DenominatorPrecision := _calculateNumeratorAndDenominatorPrecisions(decimals)

	token0SupplyExchangePrice, token0BorrowExchangePrice, err := _calcExchangePrice(token0ExchangePricesAndConfig, snapshotTimestamp)
	if err != nil {
		return CalculatedVars{}, err
	}
	token1SupplyExchangePrice, token1BorrowExchangePrice, err := _calcExchangePrice(token1ExchangePricesAndConfig, snapshotTimestamp)
	if err != nil {
		return CalculatedVars{}, err
	}

	return CalculatedVars{
		Token0NumeratorPrecision:   token0NumeratorPrecision,
		Token0DenominatorPrecision: token0DenominatorPrecision,
		Token1NumeratorPrecision:   token1NumeratorPrecision,
		Token1DenominatorPrecision: token1DenominatorPrecision,

		Token0SupplyExchangePrice: token0SupplyExchangePrice,
		Token0BorrowExchangePrice: token0BorrowExchangePrice,
		Token1SupplyExchangePrice: token1SupplyExchangePrice,
		Token1BorrowExchangePrice: token1BorrowExchangePrice,
	}, nil
}

func _calculateNumeratorAndDenominatorPrecisions(decimals int64) (*big.Int, *big.Int) {
	if decimals > TOKENS_DECIMALS_PRECISION {
		return bignumber.One, bignumber.TenPowInt(decimals - TOKENS_DECIMALS_PRECISION)
	} else {
		return bignumber.TenPowInt(TOKENS_DECIMALS_PRECISION - decimals), bignumber.One
	}
}

func _calcExchangePrice(exchangePricesAndConfig *big.Int, snapshotTimestamp uint64) (*big.Int, *big.Int, error) {
	var supplyExchangePrice, borrowExchangePrice, tmp big.Int
	supplyExchangePrice.
		Rsh(exchangePricesAndConfig, BITS_EXCHANGE_PRICES_SUPPLY_EXCHANGE_PRICE).
		And(&supplyExchangePrice, X64)

	borrowExchangePrice.
		Rsh(exchangePricesAndConfig, BITS_EXCHANGE_PRICES_BORROW_EXCHANGE_PRICE).
		And(&borrowExchangePrice, X64)

	if supplyExchangePrice.Sign() == 0 || borrowExchangePrice.Sign() == 0 {
		return nil, nil, ErrFluidLiquidityCalcsError
	}

	var temp big.Int
	temp.And(exchangePricesAndConfig, X16)

	var secondsSinceLastUpdate big.Int
	currentTime := int64(snapshotTimestamp)
	secondsSinceLastUpdate.
		Rsh(exchangePricesAndConfig, BITS_EXCHANGE_PRICES_LAST_TIMESTAMP).
		And(&secondsSinceLastUpdate, X33)
	secondsSinceLastUpdate.Sub(
		big.NewInt(currentTime),
		&secondsSinceLastUpdate,
	)

	var borrowRatio big.Int
	borrowRatio.
		Rsh(exchangePricesAndConfig, BITS_EXCHANGE_PRICES_BORROW_RATIO).
		And(&borrowRatio, X15)

	if secondsSinceLastUpdate.Sign() < 0 {
		return nil, nil, ErrFluidLiquidityCalcsError
	}
	if secondsSinceLastUpdate.Sign() == 0 || temp.Sign() == 0 || borrowRatio.Cmp(bignumber.One) == 0 {
		return &supplyExchangePrice, &borrowExchangePrice, nil
	}

	var borrowExchangePriceIncrease big.Int
	borrowExchangePriceIncrease.
		Mul(&borrowExchangePrice, &temp).
		Mul(&borrowExchangePriceIncrease, &secondsSinceLastUpdate)
	// Deployed LiquidityCalcs rounds borrow growth up; supply growth stays floored.
	tmp.Mul(SECONDS_PER_YEAR, FOUR_DECIMALS)
	borrowExchangePriceIncrease.Add(&borrowExchangePriceIncrease, &tmp).
		Sub(&borrowExchangePriceIncrease, bignumber.One).
		Div(&borrowExchangePriceIncrease, &tmp)

	borrowExchangePrice.Add(&borrowExchangePrice, &borrowExchangePriceIncrease)

	temp.Rsh(exchangePricesAndConfig, BITS_EXCHANGE_PRICES_SUPPLY_RATIO).
		And(&temp, X15)

	if temp.Cmp(bignumber.One) == 0 {
		return &supplyExchangePrice, &borrowExchangePrice, nil
	}

	if temp.Bit(0) == 1 {
		temp.Rsh(&temp, 1)
		temp.Div(
			tmp.Mul(TenPow27, FOUR_DECIMALS),
			&temp,
		)

		var utilization big.Int
		utilization.
			Rsh(exchangePricesAndConfig, BITS_EXCHANGE_PRICES_UTILIZATION).
			And(&utilization, X14).
			Mul(&utilization, tmp.Add(TenPow27, &temp))

		temp.Div(&utilization, FOUR_DECIMALS)
	} else {
		temp.Rsh(&temp, 1)
		var utilization big.Int
		utilization.
			Rsh(exchangePricesAndConfig, BITS_EXCHANGE_PRICES_UTILIZATION).
			And(&utilization, X14).
			Mul(&utilization, TenPow27).
			Mul(&utilization, tmp.Add(FOUR_DECIMALS, &temp))

		temp.Div(&utilization, tmp.Mul(FOUR_DECIMALS, FOUR_DECIMALS))
	}

	if borrowRatio.Bit(0) == 1 {
		borrowRatio.Rsh(&borrowRatio, 1).
			Mul(&borrowRatio, TenPow27).
			Div(&borrowRatio, tmp.Add(FOUR_DECIMALS, &borrowRatio))
	} else {
		borrowRatio.Rsh(&borrowRatio, 1).
			Mul(&borrowRatio, TenPow27).
			Div(&borrowRatio, tmp.Add(FOUR_DECIMALS, &borrowRatio)).
			Sub(TenPow27, &borrowRatio)
	}

	temp.Mul(&temp, FOUR_DECIMALS).
		Mul(&temp, &borrowRatio).
		Div(&temp, TenPow54)

	var borrowRate big.Int
	borrowRate.And(exchangePricesAndConfig, X16)

	var revenueFee big.Int
	revenueFee.Rsh(exchangePricesAndConfig, BITS_EXCHANGE_PRICES_FEE).
		And(&revenueFee, X14).
		Sub(FOUR_DECIMALS, &revenueFee)

	temp.
		Mul(&temp, &borrowRate).
		Mul(&temp, &revenueFee)

	var num, den big.Int
	num.Mul(&supplyExchangePrice, &temp).Mul(&num, &secondsSinceLastUpdate)
	den.Mul(SECONDS_PER_YEAR, FOUR_DECIMALS).Mul(&den, FOUR_DECIMALS).Mul(&den, FOUR_DECIMALS)

	supplyExchangePrice.Add(&supplyExchangePrice, tmp.Div(&num, &den))

	return &supplyExchangePrice, &borrowExchangePrice, nil
}

var (
	u256FourDecimals = uint256.NewInt(10_000)
	u256TenDecimals  = uint256.NewInt(10_000_000_000)
	u256Precision    = uint256.NewInt(1_000_000_000_000) // Liquidity EXCHANGE_PRICES_PRECISION (1e12)
	u256MaxSqrtMove  = uint256.NewInt(2_000_000_000)
	u256MinSqrtMove  = uint256.NewInt(5)
)

// _verifyAmountLimits mirrors the swap module's 212004 bound: [1e4, 2^128).
func _verifyAmountLimits(amount *uint256.Int) error {
	if amount.Lt(u256FourDecimals) || amount.BitLen() > 128 {
		return ErrAmountOutOfLimits
	}
	return nil
}

// _verifyAdjustedAmountLimits mirrors the swap module's 212005 bound: [1e4, 2^86).
func _verifyAdjustedAmountLimits(amount *uint256.Int) error {
	if amount.Lt(u256FourDecimals) || amount.BitLen() > 86 {
		return ErrAdjustedAmountOutOfLimits
	}
	return nil
}

// _verifySqrtPriceX96ChangeLimits mirrors 212006. Prices are below 2^160, so the
// difference times 1e10 (< 2^34) stays below 2^194.
func _verifySqrtPriceX96ChangeLimits(sqrtPriceStartX96, sqrtPriceEndX96 *uint256.Int) error {
	var change uint256.Int
	if sqrtPriceEndX96.Gt(sqrtPriceStartX96) {
		change.Sub(sqrtPriceEndX96, sqrtPriceStartX96)
	} else {
		change.Sub(sqrtPriceStartX96, sqrtPriceEndX96)
	}
	change.Mul(&change, u256TenDecimals).Div(&change, sqrtPriceStartX96)
	if change.Gt(u256MaxSqrtMove) || change.Lt(u256MinSqrtMove) {
		return ErrSqrtPriceChangeOutOfBounds
	}
	return nil
}

func extractTokenReserves(tokenReserves *big.Int) (*big.Int, *big.Int) {
	var token0Reserves, token1Reserves big.Int
	token0Reserves.Set(tokenReserves).
		Rsh(&token0Reserves, BITS_DEX_V2_TOKEN_RESERVES_TOKEN_0_RESERVES).
		And(&token0Reserves, X128)

	token1Reserves.Set(tokenReserves).
		Rsh(&token1Reserves, BITS_DEX_V2_TOKEN_RESERVES_TOKEN_1_RESERVES).
		And(&token1Reserves, X128)

	return &token0Reserves, &token1Reserves
}

func amountToAdjusted(amount, tokenNumeratorPrecision, tokenDenominatorPrecision,
	tokenSupplyExchangePrice *big.Int) *big.Int {
	var tmp1, tmp2, tmp3 big.Int
	return new(big.Int).Div(
		tmp1.Mul(
			tmp2.Mul(
				amount,
				EXCHANGE_PRICES_PRECISION,
			),
			tokenNumeratorPrecision,
		),
		tmp3.Mul(
			tokenSupplyExchangePrice,
			tokenDenominatorPrecision,
		),
	)
}

func adjustedToAmount(adjustedAmount, tokenNumeratorPrecision, tokenDenominatorPrecision,
	tokenSupplyExchangePrice *big.Int) *big.Int {
	var tmp1, tmp2, tmp3 big.Int
	return new(big.Int).Div(
		tmp1.Mul(
			tmp2.Mul(
				adjustedAmount,
				tokenDenominatorPrecision,
			),
			tokenSupplyExchangePrice,
		),
		tmp3.Mul(
			tokenNumeratorPrecision,
			EXCHANGE_PRICES_PRECISION,
		),
	)
}

// tokenVars holds one token's conversion factors between normal and raw-adjusted units.
// They depend only on the pool snapshot (decimals, exchange prices at the snapshot
// timestamp), so a simulator computes them once with the verified big.Int formulas
// above and quotes with fixed-width integers afterwards.
type tokenVars struct {
	numerator, denominator, supplyPrice, borrowPrice uint256.Int
}

func newTokenVars(numerator, denominator, supplyPrice, borrowPrice *big.Int) (v tokenVars, overflow bool) {
	overflow = v.numerator.SetFromBig(numerator) || v.denominator.SetFromBig(denominator) ||
		v.supplyPrice.SetFromBig(supplyPrice) || v.borrowPrice.SetFromBig(borrowPrice)
	return
}

func (v *tokenVars) price(d3 bool) *uint256.Int {
	if d3 {
		return &v.supplyPrice
	}
	return &v.borrowPrice
}

// toAdjusted is amountToAdjusted in fixed width: amount * 1e12 * numerator / (price * denominator).
// amount < 2^128 (checked by the caller), 1e12 < 2^40 and numerator <= 1e9 < 2^30, so the
// product stays below 2^198; price < 2^64 and denominator <= 1e9 keep the divisor below 2^94.
func (v *tokenVars) toAdjusted(dst, amount, price *uint256.Int) *uint256.Int {
	var den uint256.Int
	dst.Mul(amount, u256Precision).Mul(dst, &v.numerator)
	den.Mul(price, &v.denominator)
	return dst.Div(dst, &den)
}

// toAmount is adjustedToAmount in fixed width: adjusted * denominator * price / (numerator * 1e12).
// adjusted < 2^86 (checked by the caller), denominator <= 1e9 and price < 2^64: below 2^180.
func (v *tokenVars) toAmount(dst, adjusted, price *uint256.Int) *uint256.Int {
	var den uint256.Int
	dst.Mul(adjusted, &v.denominator).Mul(dst, price)
	den.Mul(&v.numerator, u256Precision)
	return dst.Div(dst, &den)
}
