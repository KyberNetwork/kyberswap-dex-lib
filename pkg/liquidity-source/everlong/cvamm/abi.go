package everlongcvamm

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var almABI abi.ABI

func init() {
	var err error
	almABI, err = abi.JSON(bytes.NewReader(almABIJson))
	if err != nil {
		panic(err)
	}
}
