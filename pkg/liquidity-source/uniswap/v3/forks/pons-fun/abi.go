package ponsfun

import (
	"bytes"
	_ "embed"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

//go:embed abis/PonsToken.json
var ponsTokenABIJson []byte

var ponsTokenABI abi.ABI

func init() {
	var err error
	if ponsTokenABI, err = abi.JSON(bytes.NewReader(ponsTokenABIJson)); err != nil {
		panic(err)
	}
}
