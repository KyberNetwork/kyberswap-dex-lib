package flaunch

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// positionManagerABIJson covers the two getters Track reads on the hook itself.
//
// getPoolFeeDistribution returns FeeDistributor.FeeDistribution, a static struct
// (uint24 swapFee, uint24 referrer, uint24 protocol, bool active); its ABI encoding is
// identical to four flat outputs, which is how it is declared here so the decode target
// can be a plain struct.
const positionManagerABIJson = `[
	{
		"inputs": [{"internalType": "PoolId", "name": "_poolId", "type": "bytes32"}],
		"name": "getPoolFeeDistribution",
		"outputs": [
			{"internalType": "uint24", "name": "swapFee", "type": "uint24"},
			{"internalType": "uint24", "name": "referrer", "type": "uint24"},
			{"internalType": "uint24", "name": "protocol", "type": "uint24"},
			{"internalType": "bool", "name": "active", "type": "bool"}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "feeCalculator",
		"outputs": [{"internalType": "contract IFeeCalculator", "name": "", "type": "address"}],
		"stateMutability": "view",
		"type": "function"
	}
]`

// feeCalculatorDispatcherABIJson: the v1.3+ hooks point feeCalculator at a
// FeeCalculatorDispatcher that routes each pool to a sub-calculator (zero address = the
// stateless default, a flat StaticFeeCalculator).
const feeCalculatorDispatcherABIJson = `[
	{
		"inputs": [{"internalType": "PoolId", "name": "_poolId", "type": "bytes32"}],
		"name": "poolCalculator",
		"outputs": [{"internalType": "contract IFeeCalculator", "name": "_calculator", "type": "address"}],
		"stateMutability": "view",
		"type": "function"
	}
]`

// spendGatedCalculatorABIJson: SpendGatedSignerFeeCalculator.spendGateSettings. While a gate
// is enforcing, swaps that do not carry a signed authorization revert in afterSwap, so such a
// pool cannot be routed by an aggregator.
const spendGatedCalculatorABIJson = `[
	{
		"inputs": [{"internalType": "PoolId", "name": "_poolId", "type": "bytes32"}],
		"name": "spendGateSettings",
		"outputs": [
			{"internalType": "bool", "name": "enabled", "type": "bool"},
			{"internalType": "uint256", "name": "walletCapWei", "type": "uint256"},
			{"internalType": "address", "name": "settler", "type": "address"},
			{"internalType": "uint256", "name": "endsAt", "type": "uint256"}
		],
		"stateMutability": "view",
		"type": "function"
	}
]`

var (
	positionManagerABI         abi.ABI
	feeCalculatorDispatcherABI abi.ABI
	spendGatedCalculatorABI    abi.ABI
)

func init() {
	for _, entry := range []struct {
		dst  *abi.ABI
		json string
	}{
		{&positionManagerABI, positionManagerABIJson},
		{&feeCalculatorDispatcherABI, feeCalculatorDispatcherABIJson},
		{&spendGatedCalculatorABI, spendGatedCalculatorABIJson},
	} {
		parsed, err := abi.JSON(bytes.NewReader([]byte(entry.json)))
		if err != nil {
			panic(err)
		}
		*entry.dst = parsed
	}
}
