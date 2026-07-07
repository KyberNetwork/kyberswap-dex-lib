package pamm

import (
	"bytes"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	lensABI       abi.ABI
	routerABI     abi.ABI
	multicall3ABI abi.ABI
)

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
	for _, b := range []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&lensABI, lensABIData},
		{&routerABI, routerABIData},
	} {
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
