package helper

import (
	"slices"

	"github.com/ethereum/go-ethereum/common"
	util "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/lo1inch/helper/utils"
)

type Whitelist struct {
	Addresses []util.AddressHalf
}

func (wl Whitelist) IsWhitelisted(taker common.Address) bool {
	addressHalf := util.HalfAddressFromAddress(taker)
	return slices.Contains(wl.Addresses, addressHalf)
}
