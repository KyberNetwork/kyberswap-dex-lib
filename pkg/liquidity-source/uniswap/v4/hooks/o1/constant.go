package o1

import (
	"github.com/ethereum/go-ethereum/common"
)

// HookAddresses is the single shared LaunchHook instance reused by every o1 launch
// (per-pool economics live in poolConfig, not in the hook's own address).
var HookAddresses = []common.Address{
	common.HexToAddress("0x1f91c998e7c2f4b690d75bdbf6502bdcd6e02acc"),
}
