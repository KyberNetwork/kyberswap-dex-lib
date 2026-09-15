package netstaking

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

var number_1e18 = uint256.NewInt(1e18)

// wsToSNet converts a wsNET amount to sNET: res = wsAmount * index / 1e18.
// res may alias wsAmount.
func wsToSNet(res, wsAmount, index *uint256.Int) (*uint256.Int, error) {
	if index.IsZero() {
		return nil, ErrIndexZero
	}
	return big256.MulWadDown(res, wsAmount, index), nil
}

// sNetToWs converts an sNET amount to wsNET: res = sNetAmount * 1e18 / index.
// res may alias sNetAmount.
func sNetToWs(res, sNetAmount, index *uint256.Int) (*uint256.Int, error) {
	if index.IsZero() {
		return nil, ErrIndexZero
	}
	return big256.MulDivDown(res, sNetAmount, number_1e18, index), nil
}
