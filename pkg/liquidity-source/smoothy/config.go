package smoothy

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Config struct {
	DexId valueobject.Exchange `json:"dexId"`
	Pool  common.Address       `json:"pool"`
}
