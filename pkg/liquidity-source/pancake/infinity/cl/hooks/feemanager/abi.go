package feemanager

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/samber/lo"
)

var Abi = lo.Must(abi.JSON(strings.NewReader(
	// language=json
	`[
  {
    "inputs": [],
    "name": "enableLPFeeOverride",
    "outputs": [
      {
        "internalType": "bool",
        "type": "bool"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "PoolId",
        "type": "bytes32"
      }
    ],
    "name": "lpFees",
    "outputs": [
      {
        "internalType": "uint32",
        "type": "uint32"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  }
]
`)))
