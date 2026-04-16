package lunarbase

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	ChainID     valueobject.ChainID `json:"chainID"`
	DexID       string              `json:"dexID"`
	CoreAddress string              `json:"coreAddress"`
	WsURL       string              `json:"wsURL"`
	FlashWsURL  string              `json:"flashWsURL"`
}
