//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple Vault
//msgp:shim *big.Int as:[]byte using:msgpencode.EncodeInt/msgpencode.DecodeInt
//msgp:shim common.Address as:[]byte using:(common.Address).Bytes/common.BytesToAddress

package madmex

import (
	"math/big"

	constant "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
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

	UseSwapPricing bool // currently not used, always false
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

// initialize Vault.PriceFeed when Vault is constructed via unmarshaling
func (v *Vault) initialize() error {
	return v.PriceFeed.initialize()
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

func (v *Vault) GetMinPrice(token string) (*big.Int, error) {
	return v.PriceFeed.GetPrice(token, false, v.IncludeAmmPrice, v.UseSwapPricing)
}

func (v *Vault) GetMaxPrice(token string) (*big.Int, error) {
	return v.PriceFeed.GetPrice(token, true, v.IncludeAmmPrice, v.UseSwapPricing)
}

func (v *Vault) GetTargetUSDGAmount(token string) *big.Int {
	supply := v.USDG.TotalSupply

	if supply.Cmp(constant.ZeroBI) == 0 {
		return constant.ZeroBI
	}

	weight := v.TokenWeights[token]

	return new(big.Int).Div(new(big.Int).Mul(weight, supply), v.TotalTokenWeights)
}

func (v *Vault) AdjustForDecimals(amount *big.Int, tokenDiv string, tokenMul string) *big.Int {
	var decimalsDiv *big.Int
	if tokenDiv == v.USDG.Address {
		decimalsDiv = USDGDecimals
	} else {
		decimalsDiv = v.TokenDecimals[tokenDiv]
	}

	var decimalsMul *big.Int
	if tokenMul == v.USDG.Address {
		decimalsMul = USDGDecimals
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

func (v *Vault) IncreaseUSDGAmount(token string, amount *big.Int) {
	v.USDGAmounts[token] = new(big.Int).Add(v.USDGAmounts[token], amount)
}

func (v *Vault) DecreaseUSDGAmount(token string, amount *big.Int) {
	currentUSDGAmount := v.USDGAmounts[token]

	if currentUSDGAmount.Cmp(amount) < 0 {
		v.USDGAmounts[token] = constant.ZeroBI
		return
	}

	v.USDGAmounts[token] = new(big.Int).Sub(v.USDGAmounts[token], amount)
}

func (v *Vault) IncreasePoolAmount(token string, amount *big.Int) {
	v.PoolAmounts[token] = new(big.Int).Add(v.PoolAmounts[token], amount)
}

func (v *Vault) DecreasePoolAmount(token string, amount *big.Int) {
	v.PoolAmounts[token] = new(big.Int).Sub(v.PoolAmounts[token], amount)
}
