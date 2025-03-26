package shared

import (
	"github.com/holiman/uint256"
)

func (b *ExtraBuffer) ConvertToShares(assets *uint256.Int) *uint256.Int {
	assets.MulDivOverflow(assets, b.TotalSupply, b.TotalAssets)
	return assets
}

func (b *ExtraBuffer) ConvertToAssets(shares *uint256.Int) *uint256.Int {
	shares.MulDivOverflow(shares, b.TotalAssets, b.TotalSupply)
	return shares
}
