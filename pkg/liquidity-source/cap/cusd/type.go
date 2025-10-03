package cusd

import (
	"math/big"

	"github.com/holiman/uint256"
)

type FeeDataResult struct {
	MinMintFee    *big.Int
	Slope0        *big.Int
	Slope1        *big.Int
	MintKinkRatio *big.Int
	BurnKinkRatio *big.Int
	OptimalRatio  *big.Int
}

type PriceResult struct {
	Price       *big.Int
	LastUpdated *big.Int
}

type Extra struct {
	Paused             bool           `json:"paused"`
	AssetsPaused       []bool         `json:"assetsPaused"`
	IsWhitelist        bool           `json:"isWhitelist"`
	CapSupply          *uint256.Int   `json:"capSupply"`
	Prices             []*uint256.Int `json:"prices"`
	VaultAssetSupplies []*uint256.Int `json:"vaultAssetSupplies"`
	Fees               []*FeeData     `json:"fees"`
	Assets             []string       `json:"assets"`
	AvailableBalances  []*uint256.Int `json:"availableBalances"`
}

type FeeData struct {
	MinMintFee    *uint256.Int `json:"minMintFee"`
	Slope0        *uint256.Int `json:"slope0"`
	Slope1        *uint256.Int `json:"slope1"`
	MintKinkRatio *uint256.Int `json:"mintKinkRatio"`
	BurnKinkRatio *uint256.Int `json:"burnKinkRatio"`
	OptimalRatio  *uint256.Int `json:"optimalRatio"`
}

func (f *FeeDataResult) toFeeData() *FeeData {
	return &FeeData{
		MinMintFee:    uint256.MustFromBig(f.MinMintFee),
		Slope0:        uint256.MustFromBig(f.Slope0),
		Slope1:        uint256.MustFromBig(f.Slope1),
		MintKinkRatio: uint256.MustFromBig(f.MintKinkRatio),
		BurnKinkRatio: uint256.MustFromBig(f.BurnKinkRatio),
		OptimalRatio:  uint256.MustFromBig(f.OptimalRatio),
	}
}
