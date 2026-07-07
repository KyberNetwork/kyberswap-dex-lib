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
    "name": "TAX_RATE",
    "outputs": [
      {
        "internalType": "uint256",
        "type": "uint256"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "name": "taxCurrency",
    "outputs": [
      {
        "internalType": "address",
        "type": "address"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  }
]
`)))
