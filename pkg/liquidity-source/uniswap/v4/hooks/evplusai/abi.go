package evplusai

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

func parseABI(s string) abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(s))
	if err != nil {
		panic(err)
	}
	return parsed
}

var hookABI = parseABI(`[
 {"type":"function","name":"registered","stateMutability":"view","inputs":[{"name":"id","type":"bytes32"}],"outputs":[{"type":"bool"}]},
 {"type":"function","name":"feeStateOf","stateMutability":"view","inputs":[{"name":"id","type":"bytes32"}],"outputs":[{"type":"tuple","components":[{"name":"zeroForOneFee","type":"uint24"},{"name":"oneForZeroFee","type":"uint24"},{"name":"observedAt","type":"uint64"},{"name":"expiresAt","type":"uint64"},{"name":"sequence","type":"uint64"}]}]}
]`)

var stateViewABI = parseABI(`[
 {"type":"function","name":"getSlot0","stateMutability":"view","inputs":[{"name":"poolId","type":"bytes32"}],"outputs":[{"name":"sqrtPriceX96","type":"uint160"},{"name":"tick","type":"int24"},{"name":"protocolFee","type":"uint24"},{"name":"lpFee","type":"uint24"}]}
]`)

var clockABI = parseABI(`[
 {"type":"function","name":"getCurrentBlockTimestamp","stateMutability":"view","inputs":[],"outputs":[{"type":"uint256"}]}
]`)
