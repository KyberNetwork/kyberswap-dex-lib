package helper

import (
	"slices"

	util "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/lo1inch/helper/utils"
	"github.com/ethereum/go-ethereum/common"
)

type Whitelist struct {
	Addresses []util.AddressHalf
}

func (wl Whitelist) IsWhitelisted(taker common.Address) bool {
	addressHalf := util.HalfAddressFromAddress(taker)
	return slices.Contains(wl.Addresses, addressHalf)
}
