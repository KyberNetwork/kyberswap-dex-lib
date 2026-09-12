package lunarbase

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	ChainID valueobject.ChainID `json:"chainID"`
	DexID   string              `json:"dexID"`
	Pools   []string            `json:"pools"`
	// Deprecated: state is refreshed through verified RPC snapshots.
	WsURL      string `json:"wsURL"`
	FlashWsURL string `json:"flashWsURL"`
}
