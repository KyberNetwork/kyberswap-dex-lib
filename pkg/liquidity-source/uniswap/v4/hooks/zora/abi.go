package zora

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/samber/lo"
)

// getPoolCoinByHash returns the dynamic struct PoolCoin{address coin; LpPosition[] positions}: word 0 is its
// offset and word 1 is coin. Decoding just those two head words skips the unused positions.
var hookABI = lo.Must(abi.JSON(strings.NewReader(
	// language=json
	`[{"name":"getPoolCoinByHash","type":"function","stateMutability":"view",
"inputs":[{"name":"poolKeyHash","type":"bytes32"}],
"outputs":[{"name":"offset","type":"uint256"},{"name":"coin","type":"address"}]}]`)))

var coinABI = lo.Must(abi.JSON(strings.NewReader(
	// language=json
	`[{"name":"coinType","type":"function","stateMutability":"view","inputs":[],
"outputs":[{"name":"","type":"uint8"}]},
{"name":"creationInfo","type":"function","stateMutability":"view","inputs":[],
"outputs":[{"name":"creationTimestamp","type":"uint256"},{"name":"isDeploying","type":"bool"}]}]`)))
