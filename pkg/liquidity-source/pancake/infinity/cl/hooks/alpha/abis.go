package alpha

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/samber/lo"
)

var Abi = lo.Must(abi.JSON(strings.NewReader(
	// language=json
	`[
  {
    "inputs": [
      {
        "internalType": "PoolId",
        "type": "bytes32"
      }
    ],
    "name": "poolStartedTimestamp",
    "outputs": [
      {
        "internalType": "int64",
        "type": "int64"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  }
]
`)))
