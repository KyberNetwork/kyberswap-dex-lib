package uniswapv4

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var stateViewABI abi.ABI

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&stateViewABI, stateViewABIJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
