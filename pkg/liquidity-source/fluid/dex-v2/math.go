package dexv2

import (
	"math/big"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func (p *PoolSimulator) _calculateVars() (CalculatedVars, error) {
	// NOTE: Should not mutate CalculatedVars's fields (readonly)
	token0NumeratorPrecision, token0DenominatorPrecision := _calculateNumeratorAndDenominatorPrecisions(p.token0Decimals)
	token1NumeratorPrecision, token1DenominatorPrecision := _calculateNumeratorAndDenominatorPrecisions(p.token1Decimals)

	token0SupplyExchangePrice, err := _calcSupplyExchangePrice(p.extra.Token0ExchangePricesAndConfig)
	if err != nil {
		return CalculatedVars{}, err
	}
	token1SupplyExchangePrice, err := _calcSupplyExchangePrice(p.extra.Token1ExchangePricesAndConfig)
	if err != nil {
		return CalculatedVars{}, err
	}

	return CalculatedVars{
		Token0NumeratorPrecision:   token0NumeratorPrecision,
		Token0DenominatorPrecision: token0DenominatorPrecision,
		Token1NumeratorPrecision:   token1NumeratorPrecision,
		Token1DenominatorPrecision: token1DenominatorPrecision,

		Token0SupplyExchangePrice: token0SupplyExchangePrice,
		Token1SupplyExchangePrice: token1SupplyExchangePrice,
	}, nil
}

func _calculateNumeratorAndDenominatorPrecisions(decimals int64) (*big.Int, *big.Int) {
	if decimals > TOKENS_DECIMALS_PRECISION {
		return bignumber.One, bignumber.TenPowInt(decimals - TOKENS_DECIMALS_PRECISION)
	} else {
		return bignumber.TenPowInt(TOKENS_DECIMALS_PRECISION - decimals), bignumber.One
	}
}

func _calcSupplyExchangePrice(exchangePricesAndConfig *big.Int) (*big.Int, error) {
	var supplyExchangePrice, tmp big.Int
	supplyExchangePrice.
		Rsh(exchangePricesAndConfig, BITS_EXCHANGE_PRICES_SUPPLY_EXCHANGE_PRICE).
		And(&supplyExchangePrice, X64)

	if supplyExchangePrice.Sign() == 0 {
		return nil, ErrFluidLiquidityCalcsError
	}

	var temp big.Int
	temp.And(exchangePricesAndConfig, X16)

	var secondsSinceLastUpdate big.Int
	secondsSinceLastUpdate.
		Rsh(exchangePricesAndConfig, BITS_EXCHANGE_PRICES_LAST_TIMESTAMP).
		And(&secondsSinceLastUpdate, X33)
	secondsSinceLastUpdate.Sub(
		big.NewInt(time.Now().Unix()),
		&secondsSinceLastUpdate,
	)

	var borrowRatio big.Int
	borrowRatio.
		Rsh(exchangePricesAndConfig, BITS_EXCHANGE_PRICES_BORROW_RATIO).
		And(&borrowRatio, X15)

	if secondsSinceLastUpdate.Sign() == 0 || temp.Sign() == 0 || borrowRatio.Cmp(bignumber.One) == 0 {
		return &supplyExchangePrice, nil
	}

	// Skip borrowExchangePrice calculation since we don't use it
	temp.Rsh(exchangePricesAndConfig, BITS_EXCHANGE_PRICES_SUPPLY_RATIO).
		And(&temp, X15)

	if temp.Cmp(bignumber.One) == 0 {
		return &supplyExchangePrice, nil
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

	return &supplyExchangePrice, nil
}
