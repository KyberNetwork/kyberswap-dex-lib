package caliber

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var caliberABI abi.ABI

func init() {
	var err error
	caliberABI, err = abi.JSON(bytes.NewReader(caliberABIBytes))
	if err != nil {
		panic(err)
	}
}
