package metavault

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type VaultPriceFeed struct {
	IsSecondaryPriceEnabled bool     `json:"isSecondaryPriceEnabled"`
	MaxStrictPriceDeviation *big.Int `json:"maxStrictPriceDeviation"`
	PriceSampleSpace        *big.Int `json:"priceSampleSpace"`

	PriceDecimals         map[string]*big.Int `json:"priceDecimals"`
	SpreadBasisPoints     map[string]*big.Int `json:"spreadBasisPoints"`
	AdjustmentBasisPoints map[string]*big.Int `json:"adjustmentBasisPoints"`
	StrictStableTokens    map[string]bool     `json:"strictStableTokens"`
	IsAdjustmentAdditive  map[string]bool     `json:"isAdjustmentAdditive"`

	SecondaryPriceFeedAddress common.Address `json:"-"`
	SecondaryPriceFeed        IFastPriceFeed `json:"secondaryPriceFeed"`
	SecondaryPriceFeedVersion int            `json:"secondaryPriceFeedVersion"`

	PriceFeedsAddresses map[string]common.Address `json:"-"`
	PriceFeeds          map[string]*PriceFeed     `json:"priceFeeds"`
}

func NewVaultPriceFeed() *VaultPriceFeed {
	return &VaultPriceFeed{
		PriceDecimals:         make(map[string]*big.Int),
		SpreadBasisPoints:     make(map[string]*big.Int),
		AdjustmentBasisPoints: make(map[string]*big.Int),
		StrictStableTokens:    make(map[string]bool),
		IsAdjustmentAdditive:  make(map[string]bool),
		PriceFeedsAddresses:   make(map[string]common.Address),
		PriceFeeds:            make(map[string]*PriceFeed),
	}
}

const (
	VaultPriceFeedMethodIsSecondaryPriceEnabled = "isSecondaryPriceEnabled"
	VaultPriceFeedMethodMaxStrictPriceDeviation = "maxStrictPriceDeviation"
	VaultPriceFeedMethodPriceSampleSpace        = "priceSampleSpace"
	VaultPriceFeedMethodSecondaryPriceFeed      = "secondaryPriceFeed"

	VaultPriceFeedMethodPriceFeeds            = "priceFeeds"
	VaultPriceFeedMethodPriceDecimals         = "priceDecimals"
	VaultPriceFeedMethodSpreadBasisPoints     = "spreadBasisPoints"
	VaultPriceFeedMethodAdjustmentBasisPoints = "adjustmentBasisPoints"
	VaultPriceFeedMethodStrictStableTokens    = "strictStableTokens"
	VaultPriceFeedMethodIsAdjustmentAdditive  = "isAdjustmentAdditive"
)
