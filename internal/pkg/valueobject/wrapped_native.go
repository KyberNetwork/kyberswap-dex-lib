package valueobject

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var WrappedNativeMap = valueobject.WrappedNativeMap

// WrapNativeLower wraps, if applicable, native token to wrapped token; and then lowercase it.
func WrapNativeLower(token string, chainID ChainID) string {
	return valueobject.WrapNativeLower(token, chainID)
}

func IsWrappedNative(address string, chainID ChainID) bool {
	return valueobject.IsWrappedNative(address, chainID)
}
