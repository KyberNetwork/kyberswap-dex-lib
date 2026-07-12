package ringswapbacking

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var routerABI abi.ABI

func init() {
	parsed, err := abi.JSON(bytes.NewReader(routerABIData))
	if err != nil {
		panic(err)
	}
	routerABI = parsed
}
