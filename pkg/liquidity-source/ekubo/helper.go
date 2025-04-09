package ekubo

import (
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func FromEkuboAddress(address string, chainID valueobject.ChainID) string {
	if address == valueobject.ZeroAddress {
		return strings.ToLower(valueobject.WrappedNativeMap[chainID])
	}
	return strings.ToLower(address)
}
