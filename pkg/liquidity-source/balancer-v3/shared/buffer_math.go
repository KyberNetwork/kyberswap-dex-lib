package shared

import (
	"github.com/holiman/uint256"
)

var (
	WAD = uint256.NewInt(1e18) // 10**18
)

func (b *ExtraBuffer) ConvertToShares(assets *uint256.Int) *uint256.Int {
	assets.MulDivOverflow(assets, WAD, b.Rate)

	return assets
}

func (b *ExtraBuffer) ConvertToAssets(shares *uint256.Int) *uint256.Int {
	shares.MulDivOverflow(shares, b.Rate, WAD)
	return shares
}
