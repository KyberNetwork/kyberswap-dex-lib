package aegis

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

var (
	HookAddresses = []common.Address{
		common.HexToAddress("0xa0b0d2d00fd544d8e0887f1a3cedd6e24baf10cc"),
	}
	DynamicFeeFlag = big.NewInt(0x800000)
)
