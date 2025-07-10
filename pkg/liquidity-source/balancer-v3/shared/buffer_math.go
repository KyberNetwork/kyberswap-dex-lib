package shared

import (
	"errors"

	"github.com/holiman/uint256"
)

var (
	WAD            = uint256.NewInt(1e18) // 10**18
	ErrInvalidRate = errors.New("invalid rate")
)

func (b *ExtraBuffer) ConvertToShares(assets *uint256.Int) (*uint256.Int, error) {
	if b.Rate == nil || b.Rate.IsZero() {
		return nil, ErrInvalidRate
	}

	assets.MulDivOverflow(assets, WAD, b.Rate)
	return assets, nil
}

func (b *ExtraBuffer) ConvertToAssets(shares *uint256.Int) (*uint256.Int, error) {
	if b.Rate == nil || b.Rate.IsZero() {
		return nil, ErrInvalidRate
	}
	shares.MulDivOverflow(shares, b.Rate, WAD)
	return shares, nil
}
