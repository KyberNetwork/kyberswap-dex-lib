package angletransmuter

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Config struct {
	DexID             string              `json:"dexID"`
	ChainID           valueobject.ChainID `json:"chainID"`
	Transmuter        string              `json:"transmuter"`
	StableTokenMethod string              `json:"stableTokenMethod"`
}
