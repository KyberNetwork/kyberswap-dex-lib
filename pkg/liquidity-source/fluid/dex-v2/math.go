package dexv2

import (
	"math/big"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func _calculateVars(dexVariables2, token0ExchangePricesAndConfig, token1ExchangePricesAndConfig *big.Int) (CalculatedVars, error) {
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

	token0SupplyExchangePrice, token0BorrowExchangePrice, err := _calcExchangePrice(token0ExchangePricesAndConfig)
	if err != nil {
		return CalculatedVars{}, err
	}
	token1SupplyExchangePrice, token1BorrowExchangePrice, err := _calcExchangePrice(token1ExchangePricesAndConfig)
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

func _calcExchangePrice(exchangePricesAndConfig *big.Int) (*big.Int, *big.Int, error) {
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
	currentTime := time.Now().Unix()
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

	if secondsSinceLastUpdate.Sign() == 0 || temp.Sign() == 0 || borrowRatio.Cmp(bignumber.One) == 0 {
		return &supplyExchangePrice, &borrowExchangePrice, nil
	}

	var borrowExchangePriceIncrease big.Int
	borrowExchangePriceIncrease.
		Mul(&borrowExchangePrice, &temp).
		Mul(&borrowExchangePriceIncrease, &secondsSinceLastUpdate).
		Div(&borrowExchangePriceIncrease, tmp.Mul(SECONDS_PER_YEAR, FOUR_DECIMALS))

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

func _verifyAmountLimits(amount *big.Int) error {
	if amount.Cmp(FOUR_DECIMALS) < 0 || amount.Cmp(X128) > 0 {
		return ErrAmountOutOfLimits
	}
	return nil
}

func _verifyAdjustedAmountLimits(amount *big.Int) error {
	if amount.Cmp(FOUR_DECIMALS) < 0 || amount.Cmp(X86) > 0 {
		return ErrAdjustedAmountOutOfLimits
	}
	return nil
}

func _verifySqrtPriceX96ChangeLimits(sqrtPriceStartX96, sqrtPriceEndX96 *big.Int) error {
	var percentageChange big.Int

	if sqrtPriceEndX96.Cmp(sqrtPriceStartX96) > 0 {
		percentageChange.Sub(sqrtPriceEndX96, sqrtPriceStartX96)
	} else {
		percentageChange.Sub(sqrtPriceStartX96, sqrtPriceEndX96)
	}

	percentageChange.Mul(&percentageChange, TEN_DECIMALS).
		Div(&percentageChange, sqrtPriceStartX96)

	if percentageChange.Cmp(MAX_SQRT_PRICE_CHANGE_PERCENTAGE) > 0 || percentageChange.Cmp(MIN_SQRT_PRICE_CHANGE_PERCENTAGE) < 0 {
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

func calculateDynamicFeeVariables(sqrtPriceX96 *big.Int, swap0To1 bool, dexVariables2 *big.Int) (DynamicFeeVariablesUI, error) {
	var d DynamicFeeVariables
	var tmp big.Int

	currentTime := time.Now().Unix()
	var newLastUpdateTimestamp, lastUpdateTimestamp big.Int
	newLastUpdateTimestamp.SetUint64(uint64(currentTime)).And(&newLastUpdateTimestamp, X15)
	lastUpdateTimestamp.Set(dexVariables2).
		Rsh(&lastUpdateTimestamp, BITS_DEX_V2_VARIABLES2_LAST_UPDATE_TIMESTAMP).
		And(&lastUpdateTimestamp, X15)

	var timeElapsed big.Int
	if newLastUpdateTimestamp.Cmp(&lastUpdateTimestamp) < 0 {
		timeElapsed.Add(X15, bignumber.One).
			Add(&timeElapsed, &newLastUpdateTimestamp).
			Sub(&timeElapsed, &lastUpdateTimestamp)
	} else {
		timeElapsed.Sub(&newLastUpdateTimestamp, &lastUpdateTimestamp)
	}

	var decayTimeRemaining big.Int
	decayTimeRemaining.Set(dexVariables2).
		Rsh(&decayTimeRemaining, BITS_DEX_V2_VARIABLES2_DECAY_TIME_REMAINING).
		And(&decayTimeRemaining, X12)

	var netPriceImpact, dexVariables2Int big.Int

	if timeElapsed.Cmp(&decayTimeRemaining) < 0 {
		netPriceImpact.Set(&dexVariables2Int).
			Rsh(&netPriceImpact, BITS_DEX_V2_VARIABLES2_ABSOLUTE_NET_PRICE_IMPACT).
			And(&netPriceImpact, X20)

		tmp.Set(dexVariables2).Rsh(&tmp, BITS_DEX_V2_VARIABLES2_NET_PRICE_IMPACT_SIGN).And(&tmp, bignumber.One)
		if tmp.Sign() == 0 {
			netPriceImpact.Neg(&netPriceImpact)
		}

		netPriceImpact.Mul(&netPriceImpact, tmp.Sub(&decayTimeRemaining, &timeElapsed))
		netPriceImpact.Div(&netPriceImpact, &decayTimeRemaining)
	}

	d.minFee = new(big.Int).Set(dexVariables2).Rsh(&tmp, BITS_DEX_V2_VARIABLES2_MIN_FEE).And(&tmp, X16)
	d.maxFee = new(big.Int).Set(dexVariables2).Rsh(&tmp, BITS_DEX_V2_VARIABLES2_MAX_FEE).And(&tmp, X16)
	d.priceImpactToFeeDivisionFactor = new(big.Int).Set(dexVariables2).Rsh(&tmp, BITS_DEX_V2_VARIABLES2_PRICE_IMPACT_TO_FEE_DIVISION_FACTOR).And(&tmp, X8)

	d.zeroPriceImpactPriceX96 = new(big.Int).Mul(sqrtPriceX96, sqrtPriceX96)
	d.zeroPriceImpactPriceX96.
		Div(d.zeroPriceImpactPriceX96, Q96).
		Mul(d.zeroPriceImpactPriceX96, SIX_DECIMALS).
		Div(d.zeroPriceImpactPriceX96, tmp.Add(SIX_DECIMALS, &netPriceImpact))

	var minFeeKinkPriceImpact big.Int
	minFeeKinkPriceImpact.Mul(d.minFee, d.priceImpactToFeeDivisionFactor)
	if minFeeKinkPriceImpact.Cmp(SIX_DECIMALS) > 0 {
		minFeeKinkPriceImpact.Set(SIX_DECIMALS)
	}
	if swap0To1 {
		minFeeKinkPriceImpact.Neg(&minFeeKinkPriceImpact)
	}

	var maxFeeKinkPriceImpact big.Int
	maxFeeKinkPriceImpact.Mul(d.maxFee, d.priceImpactToFeeDivisionFactor)
	if maxFeeKinkPriceImpact.Cmp(SIX_DECIMALS) > 0 {
		maxFeeKinkPriceImpact.Set(SIX_DECIMALS)
	}
	if swap0To1 {
		maxFeeKinkPriceImpact.Neg(&maxFeeKinkPriceImpact)
	}

	d.minFeeKinkPriceX96 = new(big.Int).Add(SIX_DECIMALS, &minFeeKinkPriceImpact)
	d.minFeeKinkPriceX96.Mul(d.zeroPriceImpactPriceX96, d.minFeeKinkPriceX96).Div(d.minFeeKinkPriceX96, SIX_DECIMALS)

	if d.minFeeKinkPriceX96.Cmp(MIN_PRICE_X96) < 0 {
		d.minFeeKinkPriceX96.Set(MIN_PRICE_X96)
	} else if d.minFeeKinkPriceX96.Cmp(MAX_PRICE_X96) > 0 {
		d.minFeeKinkPriceX96.Set(MAX_PRICE_X96)
	}

	if d.minFeeKinkPriceX96.Cmp(X160) < 0 {
		d.minFeeKinkSqrtPriceX96.Sqrt(tmp.Lsh(d.minFeeKinkPriceX96, 96))
	} else {
		d.minFeeKinkSqrtPriceX96.Sqrt(tmp.Lsh(d.minFeeKinkPriceX96, 84))
		d.minFeeKinkSqrtPriceX96.Lsh(d.minFeeKinkSqrtPriceX96, 6)
	}

	d.maxFeeKinkPriceX96 = new(big.Int).Add(SIX_DECIMALS, &maxFeeKinkPriceImpact)
	d.maxFeeKinkPriceX96.Mul(d.zeroPriceImpactPriceX96, d.maxFeeKinkPriceX96).Div(d.maxFeeKinkPriceX96, SIX_DECIMALS)

	if d.maxFeeKinkPriceX96.Cmp(MIN_PRICE_X96) < 0 {
		d.maxFeeKinkPriceX96.Set(MIN_PRICE_X96)
	} else if d.maxFeeKinkPriceX96.Cmp(MAX_PRICE_X96) > 0 {
		d.maxFeeKinkPriceX96.Set(MAX_PRICE_X96)
	}

	if d.maxFeeKinkPriceX96.Cmp(X160) < 0 {
		d.maxFeeKinkSqrtPriceX96.Sqrt(tmp.Lsh(d.maxFeeKinkPriceX96, 96))
	} else {
		d.maxFeeKinkSqrtPriceX96.Sqrt(tmp.Lsh(d.maxFeeKinkPriceX96, 84))
		d.maxFeeKinkSqrtPriceX96.Lsh(d.maxFeeKinkSqrtPriceX96, 6)
	}

	dui := DynamicFeeVariablesUI{
		minFee:                         big256.FromBig(d.minFee),
		maxFee:                         big256.FromBig(d.maxFee),
		priceImpactToFeeDivisionFactor: big256.FromBig(d.priceImpactToFeeDivisionFactor),
		zeroPriceImpactPriceX96:        big256.FromBig(d.zeroPriceImpactPriceX96),
		minFeeKinkPriceX96:             big256.FromBig(d.minFeeKinkPriceX96),
		minFeeKinkSqrtPriceX96:         big256.FromBig(d.minFeeKinkSqrtPriceX96),
		maxFeeKinkPriceX96:             big256.FromBig(d.maxFeeKinkPriceX96),
		maxFeeKinkSqrtPriceX96:         big256.FromBig(d.maxFeeKinkSqrtPriceX96),
	}

	return dui, nil
}
