//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple Vault
//msgp:shim *big.Int as:[]byte using:msgpencode.EncodeInt/msgpencode.DecodeInt
//msgp:shim common.Address as:[]byte using:(common.Address).Bytes/common.BytesToAddress

package quickperps

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
	USDQAmounts       map[string]*big.Int `json:"usdqAmounts"`
	MaxUSDQAmounts    map[string]*big.Int `json:"maxUsdqAmounts"`
	TokenWeights      map[string]*big.Int `json:"tokenWeights"`

	PriceFeedAddress common.Address  `json:"-"`
	PriceFeed        *VaultPriceFeed `json:"priceFeed"`

	USDQAddress common.Address `json:"-"`
	USDQ        *USDQ          `json:"usdq"`

	WhitelistedTokensCount *big.Int `json:"-"`

	UseSwapPricing bool // not used, always false for now
}

func NewVault() *Vault {
	return &Vault{
		PoolAmounts:     make(map[string]*big.Int),
		BufferAmounts:   make(map[string]*big.Int),
		ReservedAmounts: make(map[string]*big.Int),
		TokenDecimals:   make(map[string]*big.Int),
		StableTokens:    make(map[string]bool),
		USDQAmounts:     make(map[string]*big.Int),
		MaxUSDQAmounts:  make(map[string]*big.Int),
		TokenWeights:    make(map[string]*big.Int),
	}
}

// initialize Vault.PriceFeed when Vault is constructed via unmarshaling
func (v *Vault) initialize() error {
	return v.PriceFeed.initialize()
}

const (
	vaultMethodHasDynamicFees           = "hasDynamicFees"
	vaultMethodIncludeAmmPrice          = "includeAmmPrice"
	vaultMethodIsSwapEnabled            = "isSwapEnabled"
	vaultMethodPriceFeed                = "priceFeed"
	vaultMethodStableSwapFeeBasisPoints = "stableSwapFeeBasisPoints"
	vaultMethodStableTaxBasisPoints     = "stableTaxBasisPoints"
	vaultMethodSwapFeeBasisPoints       = "swapFeeBasisPoints"
	vaultMethodTaxBasisPoints           = "taxBasisPoints"
	vaultMethodTotalTokenWeights        = "totalTokenWeights"
	vaultMethodUSDQ                     = "usdq"
	vaultMethodWhitelistedTokenCount    = "whitelistedTokenCount"

	vaultMethodAllWhitelistedTokens = "allWhitelistedTokens"

	vaultMethodPoolAmounts     = "poolAmounts"
	vaultMethodBufferAmounts   = "bufferAmounts"
	vaultMethodReservedAmounts = "reservedAmounts"
	vaultMethodTokenDecimals   = "tokenDecimals"
	vaultMethodStableTokens    = "stableTokens"
	vaultMethodUSDQAmounts     = "usdqAmounts"
	vaultMethodMaxUSDQAmounts  = "maxUsdqAmounts"
	vaultMethodTokenWeights    = "tokenWeights"
)

func (v *Vault) GetMinPrice(token string) (*big.Int, error) {
	return v.PriceFeed.GetPrice(token, false, v.IncludeAmmPrice, v.UseSwapPricing)
}

func (v *Vault) GetMaxPrice(token string) (*big.Int, error) {
	return v.PriceFeed.GetPrice(token, true, v.IncludeAmmPrice, v.UseSwapPricing)
}

func (v *Vault) GetTargetUSDQAmount(token string) *big.Int {
	supply := v.USDQ.TotalSupply

	if supply.Cmp(bignumber.ZeroBI) == 0 {
		return bignumber.ZeroBI
	}

	weight := v.TokenWeights[token]

	return new(big.Int).Div(new(big.Int).Mul(weight, supply), v.TotalTokenWeights)
}

func (v *Vault) AdjustForDecimals(amount *big.Int, tokenDiv string, tokenMul string) *big.Int {
	var decimalsDiv *big.Int
	if tokenDiv == v.USDQ.Address {
		decimalsDiv = USDQDecimals
	} else {
		decimalsDiv = v.TokenDecimals[tokenDiv]
	}

	var decimalsMul *big.Int
	if tokenMul == v.USDQ.Address {
		decimalsMul = USDQDecimals
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

func (v *Vault) IncreaseUSDQAmount(token string, amount *big.Int) {
	v.USDQAmounts[token] = new(big.Int).Add(v.USDQAmounts[token], amount)
}

func (v *Vault) DecreaseUSDQAmount(token string, amount *big.Int) {
	currentUSDQAmount := v.USDQAmounts[token]

	if currentUSDQAmount.Cmp(amount) < 0 {
		v.USDQAmounts[token] = bignumber.ZeroBI
		return
	}

	v.USDQAmounts[token] = new(big.Int).Sub(v.USDQAmounts[token], amount)
}

func (v *Vault) IncreasePoolAmount(token string, amount *big.Int) {
	v.PoolAmounts[token] = new(big.Int).Add(v.PoolAmounts[token], amount)
}

func (v *Vault) DecreasePoolAmount(token string, amount *big.Int) {
	v.PoolAmounts[token] = new(big.Int).Sub(v.PoolAmounts[token], amount)
}
