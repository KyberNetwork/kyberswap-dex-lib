package noice

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

var (
	HookAddresses = []common.Address{
		common.HexToAddress("0x3e342a06f9592459D75721d6956B570F02eF2Dc0"),
	}

	ErrCannotSwapBeforeStartingTime = errors.New("cannot swap before starting time")
)
