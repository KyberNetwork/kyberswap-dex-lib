package nightlyprop

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	DexID   string              `json:"dexID"`
	ChainID valueobject.ChainID `json:"chainId"`
	// RouterAddress is the proxy that executes swaps.
	RouterAddress string `json:"routerAddress"`
	// KyberExecutor calls swapWithFees on the router.
	KyberExecutor string `json:"kyberExecutor"`
}
