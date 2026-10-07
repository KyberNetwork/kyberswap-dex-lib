package cusd

import (
	"github.com/holiman/uint256"
)

type FeeDataResult struct {
	MinMintFee    *uint256.Int
	Slope0        *uint256.Int
	Slope1        *uint256.Int
	MintKinkRatio *uint256.Int
	BurnKinkRatio *uint256.Int
	OptimalRatio  *uint256.Int
}

type PriceResult struct {
	Price       *uint256.Int
	LastUpdated *uint256.Int
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

type Meta struct {
	BlockNumber uint64 `json:"blockNumber"`
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
		MinMintFee:    f.MinMintFee,
		Slope0:        f.Slope0,
		Slope1:        f.Slope1,
		MintKinkRatio: f.MintKinkRatio,
		BurnKinkRatio: f.BurnKinkRatio,
		OptimalRatio:  f.OptimalRatio,
	}
}
