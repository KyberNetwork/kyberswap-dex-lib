package baseline

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	DexID        string              `json:"dexID"`
	ChainID      valueobject.ChainID `json:"chainId"`
	RelayAddress string              `json:"relayAddress"`
	NewPoolLimit int                 `json:"newPoolLimit"`
}
