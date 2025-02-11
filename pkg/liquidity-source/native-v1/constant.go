package nativev1

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const DexType = "native-v1"

var (
	defaultGas         = Gas{Quote: 300000}
	bps        float64 = 10000

	chainById = map[valueobject.ChainID]string{
		valueobject.ChainIDArbitrumOne:     "arbitrum",
		valueobject.ChainIDAvalancheCChain: "avalanche",
		valueobject.ChainIDBase:            "base",
		valueobject.ChainIDBSC:             "bsc",
		valueobject.ChainIDEthereum:        "ethereum",
		valueobject.ChainIDLinea:           "linea",
		valueobject.ChainIDMantle:          "mantle",
		valueobject.ChainIDPolygon:         "polygon",
		valueobject.ChainIDScroll:          "scroll",
	}
)

var (
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
