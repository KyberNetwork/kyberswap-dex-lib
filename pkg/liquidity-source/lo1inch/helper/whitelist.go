package helper

import (
	util "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/lo1inch/helper/utils"
	"github.com/ethereum/go-ethereum/common"
)

type Whitelist struct {
	Addresses []util.AddressHalf
}

func (wl Whitelist) IsWhitelisted(taker common.Address) bool {
	addressHalf := util.HalfAddressFromAddress(taker)
	for _, item := range wl.Addresses {
		if addressHalf == item {
			return true
		}
	}

	return false
}
