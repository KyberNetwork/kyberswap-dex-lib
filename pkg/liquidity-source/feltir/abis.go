package feltir

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var feltirABI abi.ABI

func init() {
	parsed, err := abi.JSON(bytes.NewReader(feltirABIData))
	if err != nil {
		panic(err)
	}
	feltirABI = parsed
}
