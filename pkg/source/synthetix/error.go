package synthetix

import "errors"

var (
	ErrNegativeRate                  = errors.New("negative rate not supported")
	ErrAmountZero                    = errors.New("amount must be greater than 0")
	ErrInvalidAtomicSwaps            = errors.New("atomic swaps must go through sUSD")
	ErrNoAtomicEquivalentForSource   = errors.New("no atomic equivalent for source")
	ErrNoAtomicEquivalentForDest     = errors.New("no atomic equivalent for dest")
	ErrResultZero                    = errors.New("result must be greater than 0")
	ErrUninitializedAtomicTwapWindow = errors.New("uninitialized atomic twap window")
	ErrDexPriceZero                  = errors.New("dex price returned 0")
	ErrAggregatorNotFound            = errors.New("aggregator not found")
	ErrNotSortedKeys                 = errors.New("not sorted keys")
	ErrInvalidObservationCardinality = errors.New("invalid observation cardinality")
	ErrInvalidPrevInitialized        = errors.New("previous observation must be initialized")
	ErrInvalidPeriod                 = errors.New("invalid period")
	ErrInvalidSrcSynth               = errors.New("src synth rate invalid")
	ErrInvalidDestSynth              = errors.New("dest synth rate invalid")
	ErrExchangeRatesTooVolatile      = errors.New("exchange rates too volatile")
	ErrAmountExceedsTotalSupply      = errors.New("amount exceeds total supply")
	ErrSrcSynthTooVolatile           = errors.New("src synth too volatile")
	ErrDestSynthTooVolatile          = errors.New("dest synth too volatile")
	ErrSurpassedVolumeLimit          = errors.New("surpassed volume limit")
	ErrInvalidLastAtomicVolume       = errors.New("invalid LastAtomicVolume")
)
