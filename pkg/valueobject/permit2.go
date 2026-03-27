package valueobject

import (
	"github.com/ethereum/go-ethereum/common"
)

var (
	permit2Default = common.HexToAddress("0x000000000022d473030f116ddee9f6b43ac78ba3")
	permit2ZkSync  = common.HexToAddress("0x0000000000225e31d15943971f47ad3022f714fa")
)

func Permit2(chainID ChainID) common.Address {
	switch chainID {
	case ChainIDZKSync:
		return permit2ZkSync
	}
	return permit2Default
}
