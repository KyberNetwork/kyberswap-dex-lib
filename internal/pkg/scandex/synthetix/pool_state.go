package synthetix

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type PoolState struct {
	BlockTimestamp      uint64                    `json:"blockTimestamp"`
	Synths              map[string]common.Address `json:"synths"`
	CurrencyKeyBySynth  map[common.Address]string `json:"currencyKeyBySynth"`
	AvailableSynthCount *big.Int                  `json:"availableSynthCount"`
	SynthsTotalSupply   map[string]*big.Int       `json:"synthsTotalSupply"`
	TotalIssuedSUSD     *big.Int                  `json:"totalIssuedSUSD"`
	CurrencyKeys        []string                  `json:"availableCurrencyKeys"`
	SUSDCurrencyKey     string                    `json:"sUSDCurrencyKey"`
	Addresses           *Addresses                `json:"addresses"`

	// SystemSettings data, will be updated by SystemSettingsReader
	SystemSettings *SystemSettings `json:"systemSettings"`

	// ExchangerWithFeeRecAlternatives data
	AtomicMaxVolumePerBlock *big.Int                `json:"atomicMaxVolumePerBlock,omitempty"`
	LastAtomicVolume        *ExchangeVolumeAtPeriod `json:"lastAtomicVolume,omitempty"`

	// ExchangeRates data, will be updated by ExchangeRatesReader / ExchangeRatesWithDexPricingReader
	AggregatorAddresses                map[string]common.Address `json:"aggregatorAddresses"`
	CurrencyKeyDecimals                map[string]uint8          `json:"currencyKeyDecimals"`
	CurrentRoundIds                    map[string]*big.Int       `json:"currentRoundIds"`
	SynthTooVolatileForAtomicExchanges map[string]bool           `json:"synthTooVolatileForAtomicExchange,omitempty"`
	DexPriceAggregatorAddress          common.Address            `json:"dexPriceAggregatorAddress,omitempty"`

	// ChainlinkDataFeed data, will be updated by ChainlinkDataFeedReader
	Aggregators map[string]*ChainlinkDataFeed `json:"aggregators"`

	// DexPriceAggregatorUniswapV3 data, will be updated by DexPriceAggregatorUniswapV3Reader
	DexPriceAggregator *DexPriceAggregatorUniswapV3 `json:"dexPriceAggregator,omitempty"`
}

func NewPoolState() *PoolState {
	return &PoolState{
		Synths:                             make(map[string]common.Address),
		CurrencyKeyBySynth:                 make(map[common.Address]string),
		SynthsTotalSupply:                  make(map[string]*big.Int),
		CurrencyKeyDecimals:                make(map[string]uint8),
		CurrentRoundIds:                    make(map[string]*big.Int),
		SynthTooVolatileForAtomicExchanges: make(map[string]bool),
		AggregatorAddresses:                make(map[string]common.Address),
		Aggregators:                        make(map[string]*ChainlinkDataFeed),
	}
}
