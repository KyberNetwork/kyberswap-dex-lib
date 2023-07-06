package metavault

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
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
	USDMAmounts       map[string]*big.Int `json:"usdmAmounts"`
	MaxUSDMAmounts    map[string]*big.Int `json:"maxUsdmAmounts"`
	TokenWeights      map[string]*big.Int `json:"tokenWeights"`

	PriceFeedAddress common.Address  `json:"-"`
	PriceFeed        *VaultPriceFeed `json:"priceFeed"`

	USDMAddress common.Address `json:"-"`
	USDM        *USDM          `json:"usdm"`

	WhitelistedTokensCount *big.Int `json:"-"`

	UseSwapPricing bool // not used for now, always false
}

func NewVault() *Vault {
	return &Vault{
		PoolAmounts:     make(map[string]*big.Int),
		BufferAmounts:   make(map[string]*big.Int),
		ReservedAmounts: make(map[string]*big.Int),
		TokenDecimals:   make(map[string]*big.Int),
		StableTokens:    make(map[string]bool),
		USDMAmounts:     make(map[string]*big.Int),
		MaxUSDMAmounts:  make(map[string]*big.Int),
		TokenWeights:    make(map[string]*big.Int),
	}
}

const (
	VaultMethodHasDynamicFees           = "hasDynamicFees"
	VaultMethodIsSwapEnabled            = "isSwapEnabled"
	VaultMethodPriceFeed                = "priceFeed"
	VaultMethodStableSwapFeeBasisPoints = "stableSwapFeeBasisPoints"
	VaultMethodStableTaxBasisPoints     = "stableTaxBasisPoints"
	VaultMethodSwapFeeBasisPoints       = "swapFeeBasisPoints"
	VaultMethodTaxBasisPoints           = "taxBasisPoints"
	VaultMethodTotalTokenWeights        = "totalTokenWeights"
	VaultMethodUSDM                     = "usdm"
	VaultMethodWhitelistedTokenCount    = "whitelistedTokenCount"

	VaultMethodAllWhitelistedTokens = "allWhitelistedTokens"

	VaultMethodPoolAmounts     = "poolAmounts"
	VaultMethodBufferAmounts   = "bufferAmounts"
	VaultMethodReservedAmounts = "reservedAmounts"
	VaultMethodTokenDecimals   = "tokenDecimals"
	VaultMethodStableTokens    = "stableTokens"
	VaultMethodUSDMAmounts     = "usdmAmounts"
	VaultMethodMaxUSDMAmounts  = "maxUsdmAmounts"
	VaultMethodTokenWeights    = "tokenWeights"
)

func (v *Vault) GetMinPrice(token string) (*big.Int, error) {
	return v.PriceFeed.GetPrice(token, false, v.IncludeAmmPrice, v.UseSwapPricing)
}

func (v *Vault) GetMaxPrice(token string) (*big.Int, error) {
	return v.PriceFeed.GetPrice(token, true, v.IncludeAmmPrice, v.UseSwapPricing)
}

func (v *Vault) GetTargetUSDMAmount(token string) *big.Int {
	supply := v.USDM.TotalSupply

	if supply.Cmp(bignumber.ZeroBI) == 0 {
		return bignumber.ZeroBI
	}

	weight := v.TokenWeights[token]

	return new(big.Int).Div(new(big.Int).Mul(weight, supply), v.TotalTokenWeights)
}

func (v *Vault) AdjustForDecimals(amount *big.Int, tokenDiv string, tokenMul string) *big.Int {
	var decimalsDiv *big.Int
	if tokenDiv == v.USDM.Address {
		decimalsDiv = USDMDecimals
	} else {
		decimalsDiv = v.TokenDecimals[tokenDiv]
	}

	var decimalsMul *big.Int
	if tokenMul == v.USDM.Address {
		decimalsMul = USDMDecimals
	} else {
		decimalsMul = v.TokenDecimals[tokenMul]
	}

	return new(big.Int).Div(
		new(big.Int).Mul(
			amount,
			new(big.Int).Exp(big.NewInt(10), decimalsMul, nil),
		),
		new(big.Int).Exp(big.NewInt(10), decimalsDiv, nil),
	)
}

func (v *Vault) IncreaseUSDMAmount(token string, amount *big.Int) {
	v.USDMAmounts[token] = new(big.Int).Add(v.USDMAmounts[token], amount)
}

func (v *Vault) DecreaseUSDMAmount(token string, amount *big.Int) {
	currentUSDMAmount := v.USDMAmounts[token]

	if currentUSDMAmount.Cmp(amount) < 0 {
		v.USDMAmounts[token] = bignumber.ZeroBI
		return
	}

	v.USDMAmounts[token] = new(big.Int).Sub(v.USDMAmounts[token], amount)
}

func (v *Vault) IncreasePoolAmount(token string, amount *big.Int) {
	v.PoolAmounts[token] = new(big.Int).Add(v.PoolAmounts[token], amount)
}

func (v *Vault) DecreasePoolAmount(token string, amount *big.Int) {
	v.PoolAmounts[token] = new(big.Int).Sub(v.PoolAmounts[token], amount)
}
