package onetoken

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// hookABIJson covers only the getters this handler reads from the OneToken hook: the
// two per-pool mappings keyed by PoolId (launchTime, poolFeeBpsOverride) and the
// three owner-settable globals (feeBps, snipeTaxBps, snipeWindow). PoolId is a
// bytes32 alias, so the mapping getters take a bytes32 key.
const hookABIJson = `[
	{"inputs":[{"internalType":"PoolId","name":"","type":"bytes32"}],"name":"launchTime","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
	{"inputs":[{"internalType":"PoolId","name":"","type":"bytes32"}],"name":"poolFeeBpsOverride","outputs":[{"internalType":"uint16","name":"","type":"uint16"}],"stateMutability":"view","type":"function"},
	{"inputs":[],"name":"feeBps","outputs":[{"internalType":"uint16","name":"","type":"uint16"}],"stateMutability":"view","type":"function"},
	{"inputs":[],"name":"snipeTaxBps","outputs":[{"internalType":"uint16","name":"","type":"uint16"}],"stateMutability":"view","type":"function"},
	{"inputs":[],"name":"snipeWindow","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"}
]`

var HookABI abi.ABI

func init() {
	var err error
	HookABI, err = abi.JSON(bytes.NewReader([]byte(hookABIJson)))
	if err != nil {
		panic(err)
	}
}
