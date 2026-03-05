package printr

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	DexId        string              `json:"dexId"`
	ChainId      valueobject.ChainID `json:"chainId"`
	PrintrAddr   string              `json:"printrAddr"`
	TokenListAPI string              `json:"tokenListAPI"`
	NewPoolLimit int                 `json:"newPoolLimit"`
}
