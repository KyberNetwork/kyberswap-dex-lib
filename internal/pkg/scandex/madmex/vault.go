package madmex

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type Vault struct {
	HasDynamicFees           bool     `json:"hasDynamicFees"`
	IncludeAmmPrice          bool     `json:"includeAmmPrice"`
	IsSwapEnabled            bool     `json:"isSwapEnabled"`
	StableSwapFeeBasisPoints *big.Int `json:"stableSwapFeeBasisPoints"`
	StableTaxBasisPoints     *big.Int `json:"stableTaxBasisPoints"`
	SwapFeeBasisPoints       *big.Int `json:"swapFeeBasisPoints"`
	TaxBasisPoints           *big.Int `json:"taxBasisPoints"`
	TotalTokenWeights        *big.Int `json:"totalTokenWeights"`

	WhitelistedTokens []string            `json:"whitelistedTokens"`
	PoolAmounts       map[string]*big.Int `json:"poolAmounts"`
	BufferAmounts     map[string]*big.Int `json:"bufferAmounts"`
	ReservedAmounts   map[string]*big.Int `json:"reservedAmounts"`
	TokenDecimals     map[string]*big.Int `json:"tokenDecimals"`
	StableTokens      map[string]bool     `json:"stableTokens"`
	USDGAmounts       map[string]*big.Int `json:"usdgAmounts"`
	MaxUSDGAmounts    map[string]*big.Int `json:"maxUsdgAmounts"`
	TokenWeights      map[string]*big.Int `json:"tokenWeights"`

	PriceFeedAddress common.Address  `json:"-"`
	PriceFeed        *VaultPriceFeed `json:"priceFeed"`

	USDGAddress common.Address `json:"-"`
	USDG        *USDG          `json:"usdg"`

	WhitelistedTokensCount *big.Int `json:"-"`
}

func NewVault() *Vault {
	return &Vault{
		PoolAmounts:     make(map[string]*big.Int),
		BufferAmounts:   make(map[string]*big.Int),
		ReservedAmounts: make(map[string]*big.Int),
		TokenDecimals:   make(map[string]*big.Int),
		StableTokens:    make(map[string]bool),
		USDGAmounts:     make(map[string]*big.Int),
		MaxUSDGAmounts:  make(map[string]*big.Int),
		TokenWeights:    make(map[string]*big.Int),
	}
}

const (
	VaultMethodHasDynamicFees           = "hasDynamicFees"
	VaultMethodIncludeAmmPrice          = "includeAmmPrice"
	VaultMethodIsSwapEnabled            = "isSwapEnabled"
	VaultMethodPriceFeed                = "priceFeed"
	VaultMethodStableSwapFeeBasisPoints = "stableSwapFeeBasisPoints"
	VaultMethodStableTaxBasisPoints     = "stableTaxBasisPoints"
	VaultMethodSwapFeeBasisPoints       = "swapFeeBasisPoints"
	VaultMethodTaxBasisPoints           = "taxBasisPoints"
	VaultMethodTotalTokenWeights        = "totalTokenWeights"
	VaultMethodUSDG                     = "usdg"
	VaultMethodWhitelistedTokenCount    = "whitelistedTokenCount"

	VaultMethodAllWhitelistedTokens = "allWhitelistedTokens"

	VaultMethodPoolAmounts     = "poolAmounts"
	VaultMethodBufferAmounts   = "bufferAmounts"
	VaultMethodReservedAmounts = "reservedAmounts"
	VaultMethodTokenDecimals   = "tokenDecimals"
	VaultMethodStableTokens    = "stableTokens"
	VaultMethodUSDGAmounts     = "usdgAmounts"
	VaultMethodMaxUSDGAmounts  = "maxUsdgAmounts"
	VaultMethodTokenWeights    = "tokenWeights"
)
