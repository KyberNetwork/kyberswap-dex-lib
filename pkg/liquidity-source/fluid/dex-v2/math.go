package dexv2

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func (p *PoolSimulator) _calculateVars() (CalculatedVars, error) {
	// NOTE: Should not mutate CalculatedVars's fields (readonly)
	token0NumeratorPrecision, token0DenominatorPrecision := _calculateNumeratorAndDenominatorPrecisions(p.Extra.DexVariables2.Token0Decimals.Int64())
	token1NumeratorPrecision, token1DenominatorPrecision := _calculateNumeratorAndDenominatorPrecisions(p.Extra.DexVariables2.Token1Decimals.Int64())

	token0SupplyExchangePrice, err := _calcSupplyExchangePrice(p.Extra.Token0ExchangePricesAndConfig)
	if err != nil {
		return CalculatedVars{}, err
	}
	token1SupplyExchangePrice, err := _calcSupplyExchangePrice(p.Extra.Token1ExchangePricesAndConfig)
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
	var supplyExchangePrice big.Int
	supplyExchangePrice.
		Rsh(exchangePricesAndConfig, BITS_EXCHANGE_PRICES_SUPPLY_EXCHANGE_PRICE).
		And(&supplyExchangePrice, X64)

	if supplyExchangePrice.Sign() == 0 {
		return nil, ErrFluidLiquidityCalcsError
	}

	var temp big.Int
	temp.And(exchangePricesAndConfig, X16)

	var borrowRatio big.Int
	borrowRatio.
		Lsh(exchangePricesAndConfig, BITS_EXCHANGE_PRICES_BORROW_RATIO).
		And(&borrowRatio, X15)

	if temp.Sign() == 0 || borrowRatio.Cmp(bignumber.One) == 0 {
		return &supplyExchangePrice, nil
	}

	// TODO implementation
	return nil, ErrFluidLiquidityCalcsError
}
