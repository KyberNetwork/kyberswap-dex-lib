package synthetix

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type Addresses struct {
	Synthetix      string `json:"synthetix"`
	Exchanger      string `json:"exchanger"`
	ExchangeRates  string `json:"exchangeRates"`
	SystemSettings string `json:"systemSettings"`
}

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

type ExchangeVolumeAtPeriod struct {
	Time   uint64   `json:"time"`
	Volume *big.Int `json:"volume"`
}

type SystemSettings struct {
	PureChainlinkPriceForAtomicSwapsEnabled map[string]bool           `json:"pureChainlinkPriceForAtomicSwapsEnabled"`
	AtomicTwapWindow                        *big.Int                  `json:"atomicTwapWindow"`
	AtomicEquivalentForDexPricingAddresses  map[string]common.Address `json:"atomicEquivalentForDexPricingAddresses"`
	AtomicEquivalentForDexPricing           map[string]Token          `json:"atomicEquivalentForDexPricing"`
	AtomicVolatilityConsiderationWindow     map[string]*big.Int       `json:"atomicVolatilityConsiderationWindow"`
	AtomicVolatilityUpdateThreshold         map[string]*big.Int       `json:"atomicVolatilityUpdateThreshold"`
	AtomicExchangeFeeRate                   map[string]*big.Int       `json:"atomicExchangeFeeRate"`
	ExchangeFeeRate                         map[string]*big.Int       `json:"exchangeFeeRate"`
	RateStalePeriod                         *big.Int                  `json:"rateStalePeriod"`
	DynamicFeeConfig                        *DynamicFeeConfig         `json:"dynamicFeeConfig"`
}

func NewSystemSettings() *SystemSettings {
	return &SystemSettings{
		PureChainlinkPriceForAtomicSwapsEnabled: make(map[string]bool),
		AtomicEquivalentForDexPricingAddresses:  make(map[string]common.Address),
		AtomicEquivalentForDexPricing:           make(map[string]Token),
		AtomicVolatilityConsiderationWindow:     make(map[string]*big.Int),
		AtomicVolatilityUpdateThreshold:         make(map[string]*big.Int),
		AtomicExchangeFeeRate:                   make(map[string]*big.Int),
		ExchangeFeeRate:                         make(map[string]*big.Int),
	}
}

type Token struct {
	Address  common.Address `json:"address"`
	Decimals uint8          `json:"decimals"`
	Symbol   string         `json:"symbol"`
}

type DynamicFeeConfig struct {
	Threshold   *big.Int `json:"threshold"`
	WeightDecay *big.Int `json:"weightDecay"`
	Rounds      *big.Int `json:"rounds"`
	MaxFee      *big.Int `json:"maxFee"`
}

func NewDynamicFeeConfig() *DynamicFeeConfig {
	return &DynamicFeeConfig{}
}

type ChainlinkDataFeed struct {
	RoundID         *big.Int             `json:"roundId"`
	Answer          *big.Int             `json:"answer"`
	StartedAt       *big.Int             `json:"startedAt"`
	UpdatedAt       *big.Int             `json:"updatedAt"`
	AnsweredInRound *big.Int             `json:"answeredInRound"`
	Answers         map[string]RoundData `json:"answers"`
}

type RoundData struct {
	RoundId         *big.Int `json:"roundId"`
	Answer          *big.Int `json:"answer"`
	StartedAt       *big.Int `json:"startedAt"`
	UpdatedAt       *big.Int `json:"updatedAt"`
	AnsweredInRound *big.Int `json:"answeredInRound"`
}

func NewChainlinkDataFeed() *ChainlinkDataFeed {
	return &ChainlinkDataFeed{
		Answers: make(map[string]RoundData),
	}
}

type Slot0 struct {
	SqrtPriceX96               *big.Int `json:"sqrtPriceX96"`
	Tick                       *big.Int `json:"tick"`
	ObservationIndex           uint16   `json:"observationIndex"`
	ObservationCardinality     uint16   `json:"observationCardinality"`
	ObservationCardinalityNext uint16   `json:"observationCardinalityNext"`
	FeeProtocol                uint8    `json:"feeProtocol"`
	Unlocked                   bool     `json:"unlocked"`
}

type OracleObservation struct {
	// the block timestamp of the observation
	BlockTimestamp uint32 `json:"blockTimestamp"`
	// the tick accumulator, i.e. tick * time elapsed since the pool was first initialized
	TickCumulative *big.Int `json:"tickCumulative"`
	// the seconds per liquidity, i.e. seconds elapsed / max(1, liquidity) since the pool was first initialized
	SecondsPerLiquidityCumulativeX128 *big.Int `json:"secondsPerLiquidityCumulativeX128"`
	// whether or not the observation is initialized
	Initialized bool `json:"initialized"`
}

type DexPriceAggregatorUniswapV3 struct {
	DefaultPoolFee         *big.Int                                `json:"defaultPoolFee"`
	UniswapV3Factory       common.Address                          `json:"uniswapV3Factory"`
	Weth                   common.Address                          `json:"weth"`
	BlockTimestamp         uint64                                  `json:"blockTimestamp"`
	OverriddenPoolForRoute map[string]common.Address               `json:"overriddenPoolForRoute"`
	UniswapV3Slot0         map[string]Slot0                        `json:"uniswapV3Slot0"`
	UniswapV3Observations  map[string]map[uint16]OracleObservation `json:"uniswapV3Observations"`
	TickCumulatives        map[string][]*big.Int                   `json:"tickCumulatives"`
}

func NewDexPriceAggregatorUniswapV3() *DexPriceAggregatorUniswapV3 {
	return &DexPriceAggregatorUniswapV3{
		OverriddenPoolForRoute: make(map[string]common.Address),
		UniswapV3Slot0:         make(map[string]Slot0),
		UniswapV3Observations:  make(map[string]map[uint16]OracleObservation),
		TickCumulatives:        make(map[string][]*big.Int),
	}
}

type Extra struct {
	PoolState *PoolState `json:"poolState"`
}
