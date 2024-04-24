//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple VaultPriceFeed
//msgp:shim *big.Int as:[]byte using:msgpencode.EncodeInt/msgpencode.DecodeInt

package fulcrom

import (
	"math/big"
)

type VaultPriceFeed struct {
	// getPrice(tokenAddress, false)
	MinPrices map[string]*big.Int `json:"minPrices"`
	// getPrice(tokenAddress, true)
	MaxPrices map[string]*big.Int `json:"maxPrices"`
}

func NewVaultPriceFeed() *VaultPriceFeed {
	return &VaultPriceFeed{
		MinPrices: make(map[string]*big.Int),
		MaxPrices: make(map[string]*big.Int),
	}
}

const (
	vaultPriceFeedMethodGetPrice = "getPrice"
)

func (pf *VaultPriceFeed) GetPrice(token string, maximise bool) (*big.Int, error) {
	var price *big.Int

	if maximise {
		price = new(big.Int).Set(pf.MaxPrices[token])
	} else {
		price = new(big.Int).Set(pf.MinPrices[token])
	}

	return price, nil
}
