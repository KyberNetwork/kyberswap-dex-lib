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
	ForwardPrice *uint256.Int `json:"forwardPrice"`
	InversePrice *uint256.Int `json:"inversePrice"`
	FeeToken0In  *uint256.Int `json:"feeToken0In"`
	FeeToken1In  *uint256.Int `json:"feeToken1In"`
	Rate01       *uint256.Int `json:"rate01"`
	Rate10       *uint256.Int `json:"rate10"`
}

type MetaInfo struct {
	BlockNumber uint64 `json:"bN"`
}
