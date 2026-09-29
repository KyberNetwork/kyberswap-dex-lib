package cashcat

import (
	"github.com/ethereum/go-ethereum/common"
)

var (
	// HookAddresses lists known CashCatHookV2 deployments by chain.
	// The hook is not upgradeable; append new deployments here as they launch.
	HookAddresses = []common.Address{
		// Robinhood (chainID 4663)
		common.HexToAddress("0x75a54357d9c78a2db19004a5fdc76c50f9242aec"),
	}
)
