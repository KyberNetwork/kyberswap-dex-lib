package prismprop

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var routerABI abi.ABI

// Subset of Prism's public Swap ABI (IEngine structs).
const routerABIJSON = `[
	{
		"name": "getSupportedPairs",
		"type": "function",
		"stateMutability": "view",
		"inputs": [],
		"outputs": [{"name": "", "type": "tuple[]", "components": [
			{"name": "tokenA", "type": "address"},
			{"name": "tokenB", "type": "address"}
		]}]
	},
	{
		"name": "getOrderBookWithParams",
		"type": "function",
		"stateMutability": "view",
		"inputs": [
			{"name": "baseToken", "type": "address"},
			{"name": "quoteToken", "type": "address"},
			{"name": "params", "type": "tuple", "components": [
				{"name": "msgSender", "type": "address"},
				{"name": "txOrigin", "type": "address"},
				{"name": "gasPrice", "type": "uint256"},
				{"name": "pathType", "type": "uint8"}
			]}
		],
		"outputs": [{"name": "book", "type": "tuple", "components": [
			{"name": "baseToken", "type": "address"},
			{"name": "quoteToken", "type": "address"},
			{"name": "snapshotBlock", "type": "uint64"},
			{"name": "bids", "type": "tuple", "components": [
				{"name": "levels", "type": "tuple[]", "components": [
					{"name": "baseAmount", "type": "uint256"},
					{"name": "quoteAmount", "type": "uint256"}
				]},
				{"name": "outputVaultBalance", "type": "uint256"},
				{"name": "minTradeSize", "type": "uint128"},
				{"name": "maxTradeSize", "type": "uint128"},
				{"name": "tif", "type": "uint64"}
			]},
			{"name": "asks", "type": "tuple", "components": [
				{"name": "levels", "type": "tuple[]", "components": [
					{"name": "baseAmount", "type": "uint256"},
					{"name": "quoteAmount", "type": "uint256"}
				]},
				{"name": "outputVaultBalance", "type": "uint256"},
				{"name": "minTradeSize", "type": "uint128"},
				{"name": "maxTradeSize", "type": "uint128"},
				{"name": "tif", "type": "uint64"}
			]}
		]}]
	}
]`

func init() {
	var err error
	routerABI, err = abi.JSON(strings.NewReader(routerABIJSON))
	if err != nil {
		panic(err)
	}
}
