package deepstateob

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	deepstateV1ABI       abi.ABI
	deepstateBookLensABI abi.ABI
)

func init() {
	var err error
	deepstateV1ABI, err = abi.JSON(bytes.NewReader(deepstateV1ABIJson))
	if err != nil {
		panic(err)
	}
	deepstateBookLensABI, err = abi.JSON(bytes.NewReader(deepstateBookLensABIJson))
	if err != nil {
		panic(err)
	}
}
