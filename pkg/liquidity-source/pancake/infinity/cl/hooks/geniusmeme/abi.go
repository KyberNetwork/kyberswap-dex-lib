package geniusmeme

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/samber/lo"
)

var Abi = lo.Must(abi.JSON(strings.NewReader(
	// language=json
	`[
  {
    "name": "hookFeeBps",
    "outputs": [
      {
        "internalType": "uint256",
        "type": "uint256"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  }
]
`)))
