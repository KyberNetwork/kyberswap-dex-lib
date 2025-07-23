package bunniv2

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

var (
	HookAddresses = []common.Address{
		common.HexToAddress("0x0010d0d5db05933fa0d9f7038d365e1541a41888"),
		common.HexToAddress("0x0000fe59823933ac763611a69c88f91d45f81888"),
	}

	HubAddress = common.HexToAddress("0x000000dceb71f3107909b1b748424349bfde5493")

	ErrNilRpcClient = errors.New("rpc client is nil")
)
