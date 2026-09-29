package lunarbase

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	ChainID valueobject.ChainID `json:"chainID"`
	DexID   string              `json:"dexID"`
	Pools   []string            `json:"pools"`
	// SingleCallSnapshot opts into a complete latest-block aggregate without
	// a post-call canonicality check. Upgrade snapshot readers before enabling:
	// these snapshots require the completeness metadata in MessagePack v3.
	SingleCallSnapshot bool `json:"singleCallSnapshot,omitempty"`
	// Deprecated: state is refreshed through verified RPC snapshots.
	WsURL      string `json:"wsURL"`
	FlashWsURL string `json:"flashWsURL"`
}
