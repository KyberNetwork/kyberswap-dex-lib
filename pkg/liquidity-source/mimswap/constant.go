package mimswap

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const DexType = "mimswap"

var mimSwapPoolAddress = map[valueobject.ChainID][]string{
	valueobject.ChainIDArbitrumOne: {
		"0x5895bff185127A01A333cBeA8e53dCf172C13F35",
		"0x236b9eE6F185Dc8B70d8bD3649F40ec37688C1Ab",
		"0x8279699D397ED22b1014fE4D08fFD7Da7B3374C0",
	},
	valueobject.ChainIDEthereum: {
		"0x95b485615c193cf75582b70ABdB08bc7172a80fe",
		"0x6f9F9ea9c06c7D928a2fFbbCc5542b18188488E9",
		"0x567402b5E442E0a631Aab1E69aDc9747BFea1561",
	},
}
