package machima

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	poolABI        abi.ABI
	clankNowABI    abi.ABI
	tokenABI       abi.ABI
	erc20ABI       abi.ABI
	swapAdapterABI abi.ABI
)

func init() {
	poolABI = mustParseABI(`[
		{
			"inputs": [],
			"name": "slot0",
			"outputs": [
				{"internalType": "uint160", "name": "sqrtPriceX96", "type": "uint160"},
				{"internalType": "int24", "name": "tick", "type": "int24"},
				{"internalType": "uint16", "name": "observationIndex", "type": "uint16"},
				{"internalType": "uint16", "name": "observationCardinality", "type": "uint16"},
				{"internalType": "uint16", "name": "observationCardinalityNext", "type": "uint16"},
				{"internalType": "uint8", "name": "feeProtocol", "type": "uint8"},
				{"internalType": "bool", "name": "unlocked", "type": "bool"}
			],
			"stateMutability": "view",
			"type": "function"
		},
		{
			"inputs": [],
			"name": "liquidity",
			"outputs": [{"internalType": "uint128", "name": "", "type": "uint128"}],
			"stateMutability": "view",
			"type": "function"
		}
	]`)

	// getTokenTax returns TaxConfig struct:
	// (uint16 buyTaxBps, uint16 sellTaxBps, address tradingTaxHandler,
	//  address protocolTaxHandler, uint16 protocolTaxBpsWeth,
	//  uint16 protocolTaxBpsUsdc, uint16 protocolTaxBpsXma, bool hasTax)
	clankNowABI = mustParseABI(`[
		{
			"inputs": [{"internalType": "address", "name": "token", "type": "address"}],
			"name": "getTokenTax",
			"outputs": [
				{
					"components": [
						{"internalType": "uint16", "name": "buyTaxBps", "type": "uint16"},
						{"internalType": "uint16", "name": "sellTaxBps", "type": "uint16"},
						{"internalType": "address", "name": "tradingTaxHandler", "type": "address"},
						{"internalType": "address", "name": "protocolTaxHandler", "type": "address"},
						{"internalType": "uint16", "name": "protocolTaxBpsWeth", "type": "uint16"},
						{"internalType": "uint16", "name": "protocolTaxBpsUsdc", "type": "uint16"},
						{"internalType": "uint16", "name": "protocolTaxBpsXma", "type": "uint16"},
						{"internalType": "bool", "name": "hasTax", "type": "bool"}
					],
					"internalType": "struct IClankNow.TaxConfig",
					"name": "",
					"type": "tuple"
				}
			],
			"stateMutability": "view",
			"type": "function"
		}
	]`)

	// poolDeploymentTime() on MachimaToken
	tokenABI = mustParseABI(`[
		{
			"inputs": [],
			"name": "poolDeploymentTime",
			"outputs": [{"internalType": "uint256", "name": "", "type": "uint256"}],
			"stateMutability": "view",
			"type": "function"
		}
	]`)

	erc20ABI = mustParseABI(`[
		{
			"inputs": [{"internalType": "address", "name": "account", "type": "address"}],
			"name": "balanceOf",
			"outputs": [{"internalType": "uint256", "name": "", "type": "uint256"}],
			"stateMutability": "view",
			"type": "function"
		}
	]`)

	swapAdapterABI = mustParseABI(`[
		{
			"inputs": [],
			"name": "xmaSellSqrtPriceLimit",
			"outputs": [{"internalType": "uint160", "name": "", "type": "uint160"}],
			"stateMutability": "view",
			"type": "function"
		}
	]`)
}

func mustParseABI(rawJSON string) abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(rawJSON))
	if err != nil {
		panic(err)
	}
	return parsed
}
