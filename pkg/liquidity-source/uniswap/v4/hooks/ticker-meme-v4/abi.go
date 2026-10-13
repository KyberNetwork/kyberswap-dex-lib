package tickermemev4

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/samber/lo"
)

var hookABI = lo.Must(abi.JSON(strings.NewReader(`[
  {
    "type": "function", "name": "launches", "stateMutability": "view",
    "inputs": [{"name": "", "type": "bytes32"}],
    "outputs": [
      {"name": "registered", "type": "bool"},
      {"name": "memecoinIsCurrency0", "type": "bool"},
      {"name": "memecoin", "type": "address"},
      {"name": "quoteToken", "type": "address"},
      {"name": "creator", "type": "address"},
      {"name": "buybackCreatorRecipient", "type": "address"},
      {"name": "protocolFeeRecipient", "type": "address"},
      {"name": "creatorTaxBps", "type": "uint16"},
      {"name": "protocolFeeShareBps", "type": "uint16"},
      {"name": "buybackBurnBps", "type": "uint16"},
      {"name": "hookFeeBps", "type": "uint16"},
      {"name": "maxInternalPriceImpactBps", "type": "uint16"},
      {"name": "buybackEnabled", "type": "bool"}
    ]
  }
]`)))
