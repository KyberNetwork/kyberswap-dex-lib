package nativev1

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = "native-v1"

	defaultGas = 177000
	bps        = 10000
)

var (
	chainById = map[valueobject.ChainID]string{}

	ErrEmptyPriceLevels                       = errors.New("empty price levels")
	ErrAmountInIsLessThanLowestPriceLevel     = errors.New("amountIn is less than lowest price level")
	ErrAmountInIsGreaterThanHighestPriceLevel = errors.New("amountIn is greater than highest price level")
	ErrAmountOutIsGreaterThanInventory        = errors.New("amountOut is greater than inventory")
)

func ChainById(chainId valueobject.ChainID) string {
	if chain, ok := chainById[chainId]; ok {
		return chain
	}
	return chainId.String()
}
