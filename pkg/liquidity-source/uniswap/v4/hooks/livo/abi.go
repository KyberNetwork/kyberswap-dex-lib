package livo

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/samber/lo"
)

var ILivoTokenABI = lo.Must(abi.JSON(strings.NewReader(
	// language=json
	`[
  {
    "inputs": [],
    "name": "graduated",
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
    "inputs": [],
    "name": "getTaxConfig",
    "outputs": [
      {
        "components": [
          {
            "internalType": "uint16",
            "name": "buyTaxBps",
            "type": "uint16"
          },
          {
            "internalType": "uint16",
            "name": "sellTaxBps",
            "type": "uint16"
          },
          {
            "internalType": "uint64",
            "name": "taxDurationSeconds",
            "type": "uint64"
          },
          {
            "internalType": "uint64",
            "name": "graduationTimestamp",
            "type": "uint64"
          }
        ],
        "internalType": "struct ILivoToken.TaxConfig",
        "name": "config",
        "type": "tuple"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  }
]
`)))
