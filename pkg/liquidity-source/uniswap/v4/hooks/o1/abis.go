package o1

import (
	"bytes"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// launchHookABIJson covers LaunchHook.sol's `poolConfig` getter. Field set/order
// differs from b20's poolConfig (see hooks/b20/abis.go): no per-recipient bps
// split fields, and creator/platformTreasury were renamed to
// currentCreator/creatorFeeRecipient. Confirmed live via `cast call
// poolConfig(bytes32)` on 2026-09-06.
const launchHookABIJson = `[
	{
		"inputs": [{"internalType": "PoolId", "name": "", "type": "bytes32"}],
		"name": "poolConfig",
		"outputs": [
			{"internalType": "bool", "name": "initialized", "type": "bool"},
			{"internalType": "bool", "name": "tokenIsCurrency0", "type": "bool"},
			{"internalType": "address", "name": "currentCreator", "type": "address"},
			{"internalType": "address", "name": "creatorFeeRecipient", "type": "address"},
			{"internalType": "uint16", "name": "baseFeeBps", "type": "uint16"},
			{"internalType": "uint16", "name": "antiSnipeStartTotalBps", "type": "uint16"},
			{"internalType": "uint32", "name": "antiSnipeWindowSeconds", "type": "uint32"},
			{"internalType": "uint48", "name": "launchTime", "type": "uint48"}
		],
		"stateMutability": "view",
		"type": "function"
	}
]`

var LaunchHookABI abi.ABI

func init() {
	var err error
	LaunchHookABI, err = abi.JSON(bytes.NewReader([]byte(launchHookABIJson)))
	if err != nil {
		panic(err)
	}
}

// poolConfigRaw is the ethrpc.Call.AddCall decode target for `poolConfig`, field
// order matching LaunchHookABI's outputs. go-ethereum's abi.Unpack fills struct
// fields positionally, so every output must stay present even though only the
// swap-pricing-relevant fields are read downstream.
type poolConfigRaw struct {
	Initialized            bool
	TokenIsCurrency0       bool
	CurrentCreator         common.Address
	CreatorFeeRecipient    common.Address
	BaseFeeBps             uint16
	AntiSnipeStartTotalBps uint16
	AntiSnipeWindowSeconds uint32
	LaunchTime             *big.Int
}
