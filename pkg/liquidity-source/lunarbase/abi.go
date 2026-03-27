package lunarbase

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

const coreABIJSON = `[
  {
    "inputs": [],
    "name": "X",
    "outputs": [{"internalType": "address", "name": "", "type": "address"}],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "Y",
    "outputs": [{"internalType": "address", "name": "", "type": "address"}],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "CONCENTRATION_ALPHA",
    "outputs": [{"internalType": "uint8", "name": "", "type": "uint8"}],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "blockDelay",
    "outputs": [{"internalType": "uint64", "name": "", "type": "uint64"}],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "concentrationK",
    "outputs": [{"internalType": "uint32", "name": "", "type": "uint32"}],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "getXReserve",
    "outputs": [{"internalType": "uint112", "name": "", "type": "uint112"}],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "getYReserve",
    "outputs": [{"internalType": "uint112", "name": "", "type": "uint112"}],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [{"internalType": "uint256", "name": "dx", "type": "uint256"}],
    "name": "quoteXToY",
    "outputs": [
      {"internalType": "uint256", "name": "dy", "type": "uint256"},
      {"internalType": "uint160", "name": "pNext", "type": "uint160"},
      {"internalType": "uint256", "name": "fee", "type": "uint256"}
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [{"internalType": "uint256", "name": "dy", "type": "uint256"}],
    "name": "quoteYToX",
    "outputs": [
      {"internalType": "uint256", "name": "dx", "type": "uint256"},
      {"internalType": "uint160", "name": "pNext", "type": "uint160"},
      {"internalType": "uint256", "name": "fee", "type": "uint256"}
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "paused",
    "outputs": [{"internalType": "bool", "name": "", "type": "bool"}],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "state",
    "outputs": [
      {"internalType": "uint160", "name": "pX96", "type": "uint160"},
      {"internalType": "uint64", "name": "fee", "type": "uint64"},
      {"internalType": "uint64", "name": "latestUpdateBlock", "type": "uint64"}
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "anonymous": false,
    "inputs": [
      {
        "components": [
          {"internalType": "uint160", "name": "pX96", "type": "uint160"},
          {"internalType": "uint48", "name": "fee", "type": "uint48"}
        ],
        "indexed": false,
        "internalType": "struct StateUpdateParameters",
        "name": "state",
        "type": "tuple"
      }
    ],
    "name": "StateUpdated",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {"indexed": false, "internalType": "uint128", "name": "reserveX", "type": "uint128"},
      {"indexed": false, "internalType": "uint128", "name": "reserveY", "type": "uint128"}
    ],
    "name": "Sync",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {"indexed": false, "internalType": "address", "name": "recipient", "type": "address"},
      {"indexed": false, "internalType": "bool", "name": "xToY", "type": "bool"},
      {"indexed": false, "internalType": "uint256", "name": "dx", "type": "uint256"},
      {"indexed": false, "internalType": "uint256", "name": "dy", "type": "uint256"},
      {"indexed": false, "internalType": "uint256", "name": "fee", "type": "uint256"}
    ],
    "name": "SwapExecuted",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {"indexed": false, "internalType": "uint32", "name": "concentrationK", "type": "uint32"}
    ],
    "name": "ConcentrationKSet",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {"indexed": false, "internalType": "uint48", "name": "blockDelay", "type": "uint48"}
    ],
    "name": "BlockDelaySet",
    "type": "event"
  }
]`

const peripheryABIJSON = `[
  {
    "inputs": [],
    "name": "pool",
    "outputs": [{"internalType": "contract ICurvePMM", "name": "", "type": "address"}],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [{
      "components": [
        {"internalType": "address", "name": "tokenIn", "type": "address"},
        {"internalType": "address", "name": "tokenOut", "type": "address"},
        {"internalType": "uint256", "name": "amountIn", "type": "uint256"}
      ],
      "internalType": "struct ICurvePMMPeriphery.QuoteParams",
      "name": "params",
      "type": "tuple"
    }],
    "name": "quoteExactIn",
    "outputs": [{"internalType": "uint256", "name": "amountOut", "type": "uint256"}],
    "stateMutability": "view",
    "type": "function"
  }
]`

const erc20ABIJSON = `[
  {
    "inputs": [{"internalType": "address", "name": "account", "type": "address"}],
    "name": "balanceOf",
    "outputs": [{"internalType": "uint256", "name": "", "type": "uint256"}],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "decimals",
    "outputs": [{"internalType": "uint8", "name": "", "type": "uint8"}],
    "stateMutability": "view",
    "type": "function"
  }
]`

var (
	coreABI      abi.ABI
	peripheryABI abi.ABI
	erc20ABI     abi.ABI
)

func init() {
	var err error

	coreABI, err = abi.JSON(strings.NewReader(coreABIJSON))
	if err != nil {
		panic(err)
	}

	peripheryABI, err = abi.JSON(strings.NewReader(peripheryABIJSON))
	if err != nil {
		panic(err)
	}

	erc20ABI, err = abi.JSON(strings.NewReader(erc20ABIJSON))
	if err != nil {
		panic(err)
	}
}
