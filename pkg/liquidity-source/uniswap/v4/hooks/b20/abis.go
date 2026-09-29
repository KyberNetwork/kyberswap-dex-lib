package b20

import (
	"bytes"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// launchHookABIJson covers the single getter this hook reads: the frozen per-pool
// economics registered by the B20 factory at launch time (LaunchHook.sol's
// `poolConfig` public mapping getter).
const launchHookABIJson = `[
	{
		"inputs": [{"internalType": "PoolId", "name": "", "type": "bytes32"}],
		"name": "poolConfig",
		"outputs": [
			{"internalType": "bool", "name": "initialized", "type": "bool"},
			{"internalType": "bool", "name": "tokenIsCurrency0", "type": "bool"},
			{"internalType": "address", "name": "creator", "type": "address"},
			{"internalType": "address", "name": "platformTreasury", "type": "address"},
			{"internalType": "uint16", "name": "baseFeeBps", "type": "uint16"},
			{"internalType": "uint16", "name": "creatorBps", "type": "uint16"},
			{"internalType": "uint16", "name": "platformBps", "type": "uint16"},
			{"internalType": "uint16", "name": "referrerBps", "type": "uint16"},
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

// poolConfigRaw is the ethrpc.Call.AddCall decode target for `poolConfig`, field order
// matching LaunchHookABI's outputs. Only the fields relevant to swap pricing are read;
// creator/platformTreasury/creatorBps/platformBps/referrerBps affect fee *distribution*
// inside the FeeEscrow, not the swap-visible fee amount, so they aren't decoded here --
// go-ethereum's abi.Unpack fills struct fields positionally and skips absent ones
// entirely, so a struct with fewer fields than outputs does not decode correctly; instead
// every output is kept, unused ones just aren't read downstream.
type poolConfigRaw struct {
	Initialized            bool
	TokenIsCurrency0       bool
	Creator                common.Address
	PlatformTreasury       common.Address
	BaseFeeBps             uint16
	CreatorBps             uint16
	PlatformBps            uint16
	ReferrerBps            uint16
	AntiSnipeStartTotalBps uint16
	AntiSnipeWindowSeconds uint32
	LaunchTime             *big.Int
}
