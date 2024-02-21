package bancorv3

import vo "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	DexID         string     `json:"dexID"`
	ChainID       vo.ChainID `json:"chainID"`
	BancorNetwork string     `json:"bancorNetwork"`
	BNT           string     `json:"bnt"`
}
