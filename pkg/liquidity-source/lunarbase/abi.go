package lunarbase

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// coreABIJSON describes the on-chain `fix/incident` Pool surface used by
// this adapter (single-price Q32.48, asymmetric Q24 fees). The contract's
// `anchorPrice` field is what we cache as `SqrtPriceX48`.
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
    "name": "blockDelay",
    "outputs": [{"internalType": "uint48", "name": "", "type": "uint48"}],
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
      {"internalType": "uint80", "name": "pNext", "type": "uint80"},
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
      {"internalType": "uint80", "name": "pNext", "type": "uint80"},
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
      {"internalType": "uint80",  "name": "anchorPrice",       "type": "uint80"},
      {"internalType": "uint24",  "name": "feeAskX24",         "type": "uint24"},
      {"internalType": "uint24",  "name": "feeBidX24",         "type": "uint24"},
      {"internalType": "uint48",  "name": "latestUpdateBlock", "type": "uint48"}
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "anchorPrice",
    "outputs": [{"internalType": "uint80", "name": "anchorPrice", "type": "uint80"}],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "anonymous": false,
    "inputs": [
      {"indexed": false, "internalType": "uint80", "name": "anchorPrice", "type": "uint80"},
      {"indexed": false, "internalType": "uint24", "name": "feeAskX24",   "type": "uint24"},
      {"indexed": false, "internalType": "uint24", "name": "feeBidX24",   "type": "uint24"}
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
      {"indexed": false, "internalType": "bool",    "name": "xToY",      "type": "bool"},
      {"indexed": false, "internalType": "uint256", "name": "dx",        "type": "uint256"},
      {"indexed": false, "internalType": "uint256", "name": "dy",        "type": "uint256"},
      {"indexed": false, "internalType": "uint256", "name": "fee",       "type": "uint256"}
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

var (
	coreABI abi.ABI
)

func init() {
	var err error

	coreABI, err = abi.JSON(strings.NewReader(coreABIJSON))
	if err != nil {
		panic(err)
	}
}
