package carbon

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Config struct {
	DexId      valueobject.Exchange `json:"dexId"`
	ChainId    valueobject.ChainID  `json:"chainId"`
	Controller common.Address       `json:"controller"`
}
