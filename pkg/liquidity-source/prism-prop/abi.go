package prismprop

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var routerABI abi.ABI

const routerABIJSON = `[
	{
		"name": "getSupportedPairs",
		"type": "function",
		"stateMutability": "view",
		"inputs": [],
		"outputs": [{"name": "", "type": "tuple[]", "components": [
			{"name": "token0", "type": "address"},
			{"name": "token1", "type": "address"}
		]}]
	},
	{
		"name": "getOrderBook",
		"type": "function",
		"stateMutability": "nonpayable",
		"inputs": [
			{"name": "tokenSell", "type": "address"},
			{"name": "tokenBuy", "type": "address"}
		],
		"outputs": [{"name": "book", "type": "tuple", "components": [
			{"name": "token0", "type": "address"},
			{"name": "token1", "type": "address"},
			{"name": "blockNumber", "type": "uint256"},
			{"name": "side0", "type": "tuple", "components": [
				{"name": "orders", "type": "tuple[]", "components": [
					{"name": "amountIn", "type": "uint256"},
					{"name": "amountOut", "type": "uint256"}
				]},
				{"name": "s1", "type": "uint256"},
				{"name": "s2", "type": "uint256"},
				{"name": "s3", "type": "uint256"},
				{"name": "s4", "type": "uint256"}
			]},
			{"name": "side1", "type": "tuple", "components": [
				{"name": "orders", "type": "tuple[]", "components": [
					{"name": "amountIn", "type": "uint256"},
					{"name": "amountOut", "type": "uint256"}
				]},
				{"name": "s1", "type": "uint256"},
				{"name": "s2", "type": "uint256"},
				{"name": "s3", "type": "uint256"},
				{"name": "s4", "type": "uint256"}
			]}
		]}]
	},
	{
		"name": "getAmountOut",
		"type": "function",
		"stateMutability": "view",
		"inputs": [
			{"name": "tokenIn", "type": "address"},
			{"name": "tokenOut", "type": "address"},
			{"name": "amountIn", "type": "uint256"}
		],
		"outputs": [{"name": "amountOut", "type": "uint256"}]
	},
	{
		"name": "getAmountIn",
		"type": "function",
		"stateMutability": "view",
		"inputs": [
			{"name": "tokenIn", "type": "address"},
			{"name": "tokenOut", "type": "address"},
			{"name": "amountOut", "type": "uint256"}
		],
		"outputs": [{"name": "amountIn", "type": "uint256"}]
	}
]`

func init() {
	var err error
	routerABI, err = abi.JSON(strings.NewReader(routerABIJSON))
	if err != nil {
		panic(err)
	}
}
