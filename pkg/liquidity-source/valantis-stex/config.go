package valantisstex

import (
	"github.com/ethereum/go-ethereum/common"
)

type Config struct {
	DexId          string           `json:"dexId"`
	SovereignPools []common.Address `json:"sovereignPools"`
}
