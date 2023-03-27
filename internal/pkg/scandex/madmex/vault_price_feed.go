package madmex

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type VaultPriceFeed struct {
	BNB                        string   `json:"bnb"`
	BTC                        string   `json:"btc"`
	ETH                        string   `json:"eth"`
	FavorPrimaryPrice          bool     `json:"favorPrimaryPrice"`
	IsAmmEnabled               bool     `json:"isAmmEnabled"`
	IsSecondaryPriceEnabled    bool     `json:"isSecondaryPriceEnabled"`
	MaxStrictPriceDeviation    *big.Int `json:"maxStrictPriceDeviation"`
	PriceSampleSpace           *big.Int `json:"priceSampleSpace"`
	SpreadThresholdBasisPoints *big.Int `json:"spreadThresholdBasisPoints"`
	UseV2Pricing               bool     `json:"useV2Pricing"`

	PriceDecimals         map[string]*big.Int `json:"priceDecimals"`
	SpreadBasisPoints     map[string]*big.Int `json:"spreadBasisPoints"`
	AdjustmentBasisPoints map[string]*big.Int `json:"adjustmentBasisPoints"`
	StrictStableTokens    map[string]bool     `json:"strictStableTokens"`
	IsAdjustmentAdditive  map[string]bool     `json:"isAdjustmentAdditive"`

	BNBBUSDAddress common.Address `json:"-"`
	BNBBUSD        *PancakePair   `json:"bnbBusd,omitempty"`

	BTCBNBAddress common.Address `json:"-"`
	BTCBNB        *PancakePair   `json:"btcBnb,omitempty"`

	ETHBNBAddress common.Address `json:"-"`
	ETHBNB        *PancakePair   `json:"ethBnb,omitempty"`

	ChainlinkFlagsAddress common.Address  `json:"-"`
	ChainlinkFlags        *ChainlinkFlags `json:"chainlinkFlags,omitempty"`

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
	VaultPriceFeedMethodBNB                        = "bnb"
	VaultPriceFeedMethodBNBBUSD                    = "bnbBusd"
	VaultPriceFeedMethodBTC                        = "btc"
	VaultPriceFeedMethodBTCBNB                     = "btcBnb"
	VaultPriceFeedMethodChainlinkFlags             = "chainlinkFlags"
	VaultPriceFeedMethodETH                        = "eth"
	VaultPriceFeedMethodETHBNB                     = "ethBnb"
	VaultPriceFeedMethodFavorPrimaryPrice          = "favorPrimaryPrice"
	VaultPriceFeedMethodIsAmmEnabled               = "isAmmEnabled"
	VaultPriceFeedMethodIsSecondaryPriceEnabled    = "isSecondaryPriceEnabled"
	VaultPriceFeedMethodMaxStrictPriceDeviation    = "maxStrictPriceDeviation"
	VaultPriceFeedMethodPriceSampleSpace           = "priceSampleSpace"
	VaultPriceFeedMethodSecondaryPriceFeed         = "secondaryPriceFeed"
	VaultPriceFeedMethodSpreadThresholdBasisPoints = "spreadThresholdBasisPoints"
	VaultPriceFeedMethodUseV2Pricing               = "useV2Pricing"

	VaultPriceFeedMethodPriceFeeds            = "priceFeeds"
	VaultPriceFeedMethodPriceDecimals         = "priceDecimals"
	VaultPriceFeedMethodSpreadBasisPoints     = "spreadBasisPoints"
	VaultPriceFeedMethodAdjustmentBasisPoints = "adjustmentBasisPoints"
	VaultPriceFeedMethodStrictStableTokens    = "strictStableTokens"
	VaultPriceFeedMethodIsAdjustmentAdditive  = "isAdjustmentAdditive"
)
