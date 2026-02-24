package liquidcore

import (
	"github.com/holiman/uint256"
)

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
