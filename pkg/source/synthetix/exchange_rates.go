package synthetix

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// =============================================================================================
// Implementation of this contract:
// https://github.com/Synthetixio/synthetix/blob/b04a4d2948f3a575bfe8186e99086e50dc54ef95/contracts/ExchangeRates.sol

type ExchangeRates struct {
	BlockTimestamp      uint64                        `json:"blockTimestamp"`
	SUSDCurrencyKey     string                        `json:"sUSDCurrencyKey"`
	SystemSettings      *SystemSettings               `json:"systemSettings"`
	Aggregators         map[string]*ChainlinkDataFeed `json:"aggregators"`
	CurrencyKeyDecimals map[string]uint8              `json:"currencyKeyDecimals"`
}

func NewExchangeRates(
	blockTimestamp uint64,
	sUSDCurrencyKey string,
	systemSettings *SystemSettings,
	aggregators map[string]*ChainlinkDataFeed,
	currencyKeyDecimals map[string]uint8,
) *ExchangeRates {
	return &ExchangeRates{
		BlockTimestamp:      blockTimestamp,
		SUSDCurrencyKey:     sUSDCurrencyKey,
		SystemSettings:      systemSettings,
		Aggregators:         aggregators,
		CurrencyKeyDecimals: currencyKeyDecimals,
	}
}

func (er *ExchangeRates) getCurrentRoundId(currencyKey string) (*big.Int, error) {
	return er._getCurrentRoundId(currencyKey)
}

func (er *ExchangeRates) effectiveValueAndRates(
	sourceCurrencyKey string,
	sourceAmount *big.Int,
	destinationCurrencyKey string,
) (value *big.Int, sourceRate *big.Int, destinationRate *big.Int, err error) {
	return er._effectiveValueAndRates(sourceCurrencyKey, sourceAmount, destinationCurrencyKey)
}

// @notice getting N rounds of rates for a currency at a specific round
// @param currencyKey the currency key
// @param numRounds the number of rounds to get
// @param roundId the round id
// @return a list of rates and a list of times
func (er *ExchangeRates) ratesAndUpdatedTimeForCurrencyLastNRounds(
	currencyKey string,
	numRounds uint,
	roundId *big.Int,
) (rates []*big.Int, times []*big.Int, err error) {
	rates = make([]*big.Int, numRounds)
	times = make([]*big.Int, numRounds)

	if roundId.Cmp(bignumber.ZeroBI) <= 0 {
		roundId, err = er._getCurrentRoundId(currencyKey)
		if err != nil {
			return
		}
	}

	for i := uint(0); i < numRounds; i++ {
		// fetch the rate and treat is as current, so inverse limits if frozen will always be applied
		// regardless of current rate
		rates[i], times[i], err = er._getRateAndTimestampAtRound(currencyKey, roundId)
		if err != nil {
			return
		}

		if roundId.Cmp(bignumber.ZeroBI) == 0 {
			// if we hit the last round, then return what we have
			return rates, times, nil
		} else {
			new(big.Int).Sub(roundId, big.NewInt(1))
		}
	}

	return
}

func (er *ExchangeRates) rateIsInvalid(currencyKey string) bool {
	_, invalid := er.rateAndInvalid(currencyKey)

	return invalid
}

func (er *ExchangeRates) rateAndInvalid(currencyKey string) (*big.Int, bool) {
	rateAndTime, err := er._getRateAndUpdatedTime(currencyKey)
	if err != nil {
		return nil, true
	}

	if currencyKey == er.SUSDCurrencyKey {
		return rateAndTime.rate, false
	}

	return rateAndTime.rate, _rateIsStaleWithTime(
		uint(er.getRateStalePeriod().Int64()),
		uint(rateAndTime.time.Int64()),
		uint(er.BlockTimestamp),
	)

	// TODO: implement the rest here
	//return rateAndTime.rate, _rateIsStaleWithTime(uint(er.getRateStalePeriod().Int64()), uint(rateAndTime.time.Int64())) ||
	//	_rateIsFlagged(currencyKey, FlagsInterface(getAggregatorWarningFlags())) ||
	//	_rateIsCircuitBroken(currencyKey, rateAndTime.rate)

}

func (er *ExchangeRates) _formatAggregatorAnswer(currencyKey string, rate *big.Int) (*big.Int, error) {
	if rate.Cmp(bignumber.ZeroBI) < 0 {
		return nil, ErrNegativeRate
	}

	decimals := er.CurrencyKeyDecimals[currencyKey]
	result := rate
	if decimals == 0 || decimals == 18 {
		// do not convert for 0 (part of implicit interface), and not needed for 18
	} else if decimals < 18 {
		// increase precision to 18
		multiplier := bignumber.TenPowInt(18 - decimals)
		result = new(big.Int).Mul(result, multiplier)
	} else if decimals > 18 {
		// decrease precision to 18
		divisor := bignumber.TenPowInt(decimals - 18)
		result = new(big.Int).Div(result, divisor)
	}

	return result, nil
}

func (er *ExchangeRates) _getRateAndUpdatedTime(currencyKey string) (RateAndUpdatedTime, error) {
	// sUSD rate is 1.0
	if currencyKey == er.SUSDCurrencyKey {
		return RateAndUpdatedTime{
			rate: bignumber.TenPowInt(18),
			time: bignumber.ZeroBI,
		}, nil
	} else {
		aggregator := er.Aggregators[currencyKey]
		if aggregator == nil {
			return RateAndUpdatedTime{}, ErrAggregatorNotFound
		}

		answer := aggregator.LatestAnswer()
		updatedAt := aggregator.LatestUpdatedAt()

		rate, err := er._formatAggregatorAnswer(currencyKey, answer)
		if err != nil {
			return RateAndUpdatedTime{}, err
		}

		return RateAndUpdatedTime{
			rate: rate,
			time: updatedAt,
		}, nil

	}
}

func (er *ExchangeRates) _getCurrentRoundId(currencyKey string) (*big.Int, error) {
	if currencyKey == er.SUSDCurrencyKey {
		return bignumber.ZeroBI, nil
	}

	aggregator := er.Aggregators[currencyKey]
	if aggregator == nil {
		return nil, ErrAggregatorNotFound
	}

	return aggregator.LatestRound(), nil
}

func (er *ExchangeRates) _getRateAndTimestampAtRound(currencyKey string, roundId *big.Int) (*big.Int, *big.Int, error) {
	// short circuit sUSD
	if currencyKey == er.SUSDCurrencyKey {
		// sUSD has no rounds, and 0 time is preferrable for "volatility" heuristics
		// which are used in atomic swaps and fee reclamation
		return unit(), bignumber.ZeroBI, nil
	}

	aggregator := er.Aggregators[currencyKey]
	if aggregator == nil {
		return nil, nil, ErrAggregatorNotFound
	}

	_, answer, _, updatedAt, _ := aggregator.GetRoundData(roundId)

	rate, err := er._formatAggregatorAnswer(currencyKey, answer)
	if err != nil {
		return nil, nil, err
	}

	return rate, updatedAt, nil
}

func _rateIsStaleWithTime(_rateStalePeriod uint, _time uint, now uint) bool {
	return _time+_rateStalePeriod < now
}

func (er *ExchangeRates) _getRate(currencyKey string) (*big.Int, error) {
	rateAndUpdatedTime, err := er._getRateAndUpdatedTime(currencyKey)
	if err != nil {
		return nil, err
	}

	return rateAndUpdatedTime.rate, nil
}

func (er *ExchangeRates) _effectiveValueAndRates(
	sourceCurrencyKey string,
	sourceAmount *big.Int,
	destinationCurrencyKey string,
) (value *big.Int, sourceRate *big.Int, destinationRate *big.Int, err error) {
	sourceRate, err = er._getRate(sourceCurrencyKey)
	if err != nil {
		return
	}

	// If there's no change in the currency, then just return the amount they gave us
	if sourceCurrencyKey == destinationCurrencyKey {
		destinationRate = sourceRate
		value = sourceAmount
	} else {
		// Calculate the effective value by going from source -> USD -> destination
		destinationRate, err = er._getRate(destinationCurrencyKey)
		if err != nil {
			return
		}

		// prevent divide-by 0 error (this happens if the dest is not a valid rate)
		if destinationRate.Cmp(bignumber.ZeroBI) > 0 {
			value = divideDecimalRound(multiplyDecimalRound(sourceAmount, sourceRate), destinationRate)
		}
	}

	return
}

func (er *ExchangeRates) getRateStalePeriod() *big.Int {
	return er.SystemSettings.RateStalePeriod
}
