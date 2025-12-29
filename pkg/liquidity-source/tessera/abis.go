package tessera

import (
	"bytes"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	tesseraIndexerABI abi.ABI
	tesseraEngineABI  abi.ABI
	tesseraPoolABI    abi.ABI
	tesseraRouterABI  abi.ABI
	erc20ABI          abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&erc20ABI, erc20ABIData},
	}

	var err error

	for _, b := range builder {
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}

	tesseraIndexerABI, err = abi.JSON(strings.NewReader(`[
		{"inputs": [], "name": "getTesseraPairs", "outputs": [{"internalType": "address[][]", "name": "", "type": "address[][]"}], "stateMutability": "view", "type": "function"}
	]`))
	if err != nil {
		panic(err)
	}

	tesseraEngineABI, err = abi.JSON(strings.NewReader(`[
		{"inputs": [{"internalType": "address", "name": "token0", "type": "address"}, {"internalType": "address", "name": "token1", "type": "address"}], "name": "getTesseraPool", "outputs": [{"internalType": "bool", "name": "exists", "type": "bool"}, {"internalType": "address", "name": "pool", "type": "address"}], "stateMutability": "view", "type": "function"},
		{"inputs": [], "name": "getSkipLevelsBlock", "outputs": [{"internalType": "uint256", "name": "", "type": "uint256"}], "stateMutability": "view", "type": "function"}
	]`))
	if err != nil {
		panic(err)
	}

	tesseraPoolABI, err = abi.JSON(strings.NewReader(`[
		{"inputs": [], "name": "tradingEnabled", "outputs": [{"internalType": "bool", "name": "", "type": "bool"}], "stateMutability": "view", "type": "function"},
		{"inputs": [], "name": "isInitialised", "outputs": [{"internalType": "bool", "name": "", "type": "bool"}], "stateMutability": "view", "type": "function"},
		{"inputs": [], "name": "poolState", "outputs": [
			{"internalType": "uint256", "name": "poolOffset0", "type": "uint256"},
			{"internalType": "uint256", "name": "poolOffset1", "type": "uint256"},
			{"internalType": "uint32", "name": "lpFeeRate", "type": "uint32"},
			{"internalType": "uint32", "name": "mtFeeRate", "type": "uint32"},
			{"internalType": "uint8", "name": "side", "type": "uint8"},
			{"internalType": "bool", "name": "tradingEnabled", "type": "bool"},
			{"internalType": "uint64", "name": "startBlock", "type": "uint64"},
			{"internalType": "uint64", "name": "decayDuration", "type": "uint64"},
			{"internalType": "uint32", "name": "initialFeeRate", "type": "uint32"},
			{"internalType": "uint32", "name": "minimumFeeRate", "type": "uint32"},
			{"internalType": "uint32", "name": "tesseraAnchorPrice", "type": "uint32"},
			{"internalType": "bool", "name": "isWhitelistActive", "type": "bool"},
			{"internalType": "uint32", "name": "whitelistFeeRate", "type": "uint32"},
			{"internalType": "uint32", "name": "liquidatorFeeRate", "type": "uint32"},
			{"components": [{"name": "amount", "type": "uint256"}, {"name": "price", "type": "uint256"}, {"name": "active", "type": "uint256"}], "name": "orderBook0", "type": "tuple[20]"},
			{"components": [{"name": "amount", "type": "uint256"}, {"name": "price", "type": "uint256"}, {"name": "active", "type": "uint256"}], "name": "orderBook1", "type": "tuple[20]"}
		], "stateMutability": "view", "type": "function"}
	]`))
	if err != nil {
		panic(err)
	}

	tesseraRouterABI, err = abi.JSON(strings.NewReader(`[
		{"inputs": [{"internalType": "address", "name": "tokenIn", "type": "address"}, {"internalType": "address", "name": "tokenOut", "type": "address"}, {"internalType": "int256", "name": "amountSpecified", "type": "int256"}], "name": "tesseraSwapViewAmounts", "outputs": [{"internalType": "uint256", "name": "amountIn", "type": "uint256"}, {"internalType": "uint256", "name": "amountOut", "type": "uint256"}], "stateMutability": "view", "type": "function"}
	]`))
	if err != nil {
		panic(err)
	}
}
