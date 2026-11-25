package nightlyprop

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Config struct {
	DexID   string              `json:"dexID"`
	ChainID valueobject.ChainID `json:"chainId"`
	// RouterAddress is the proxy that executes swaps.
	RouterAddress string `json:"routerAddress"`
	// KyberExecutor calls swapWithFees on the router.
	KyberExecutor string `json:"kyberExecutor"`
}

func parseNonzeroAddress(raw string) (common.Address, bool) {
	if !common.IsHexAddress(raw) {
		return common.Address{}, false
	}
	address := common.HexToAddress(raw)
	return address, !valueobject.IsZeroAddress(address)
}
