package nativev1

import (
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

func ChainById(chainId valueobject.ChainID) string {
	if chain, ok := chainById[chainId]; ok {
		return chain
	}
	return chainId.String()
}
