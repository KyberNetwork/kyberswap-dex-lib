package dexT1

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

const (
	DexType = "fluid-dex-t1"
)

var dexReservesResolver = map[valueobject.ChainID]string{
	valueobject.ChainIDEthereum: "0x278166A9B88f166EB170d55801bE1b1d1E576330",
}

const (
	// DexReservesResolver methods
	DRRMethodGetAllPoolsReserves = "getAllPoolsReserves"
	DRRMethodGetPoolReserves     = "getPoolReserves"
)

const Fee100PercentPrecision int64 = 1e6
