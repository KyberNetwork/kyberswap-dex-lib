package liquidcore

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type Metadata struct {
	LastCount         int            `json:"count"`
	LastPoolsChecksum common.Address `json:"poolsChecksum"`
}

type Extra struct {
	SpotPrice   *uint256.Int `json:"spotPrice"`
	OraclePrice *uint256.Int `json:"oraclePrice"`
}

type MetaInfo struct {
	BlockNumber uint64 `json:"bN"`
}
