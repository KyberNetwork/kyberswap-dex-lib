package synthetix

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

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
