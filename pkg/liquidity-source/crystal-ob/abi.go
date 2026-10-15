package crystalob

import (
	"bytes"
	_ "embed"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

//go:embed abis/Crystal.json
var crystalABIJson []byte

var crystalABI = func() abi.ABI {
	parsed, err := abi.JSON(bytes.NewReader(crystalABIJson))
	if err != nil {
		panic(err)
	}
	return parsed
}()
