package valueobject

import (
	"time"

	"github.com/KyberNetwork/logger"
)

type SubgraphMeta struct {
	Block struct {
		Timestamp int64 `json:"timestamp"`
	} `json:"block"`
}

func (m *SubgraphMeta) CheckIsLagging(names ...string) {
	if m != nil {
		now := time.Now().Unix()
		lag := now - m.Block.Timestamp

		// hardcode threshold to 10min for now, this will soon be replaced with pool-ticks
		if lag > 60*10 {
			logger.Warnf("subgraph is lagging by %v seconds for %v", lag, names)
		}
	} else {
		logger.Warnf("subgraph meta is empty for %v", names)
	}
}
