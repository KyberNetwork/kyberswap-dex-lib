package dexT1

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

const (
	DexType = "fluid-dex-t1"
)

var dexReservesResolver = map[valueobject.ChainID]string{
	valueobject.ChainIDEthereum: "0xfE1CBE632855e279601EaAF58D3cB552271BfDF5",
}

const (
	// DexReservesResolver methods
	DRRMethodGetAllPoolsReserves = "getAllPoolsReserves"
)
