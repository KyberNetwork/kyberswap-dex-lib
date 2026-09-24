package prop

import (
	"bytes"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	lensABI        abi.ABI
	swapABI        abi.ABI
	lensPammABI    abi.ABI
	routerPammABI  abi.ABI
	positionCapABI abi.ABI
	multicall3ABI  abi.ABI
)

// multicall3ABIJSON covers only tryAggregate, the single method the pamm
// variant's probing needs.
const multicall3ABIJSON = `[{
  "name": "tryAggregate",
  "type": "function",
  "stateMutability": "payable",
  "inputs": [
    {"name": "requireSuccess", "type": "bool"},
    {"name": "calls", "type": "tuple[]", "components": [
      {"name": "target", "type": "address"},
      {"name": "callData", "type": "bytes"}
    ]}
  ],
  "outputs": [
    {"name": "returnData", "type": "tuple[]", "components": [
      {"name": "success", "type": "bool"},
      {"name": "returnData", "type": "bytes"}
    ]}
  ]
}]`

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&lensABI, lensABIData},
		{&swapABI, swapABIData},
		{&lensPammABI, lensPammABIData},
		{&routerPammABI, routerPammABIData},
		{&positionCapABI, positionCapABIData},
	}

	for _, b := range builder {
		parsed, err := abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
		*b.ABI = parsed
	}

	parsed, err := abi.JSON(strings.NewReader(multicall3ABIJSON))
	if err != nil {
		panic(err)
	}
	multicall3ABI = parsed
}
