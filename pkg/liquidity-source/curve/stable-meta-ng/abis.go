package stablemetang

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	curveStableMetaNGABI abi.ABI
)

func init() {
	var build = []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&curveStableMetaNGABI, curveStableMetaNGABIBytes},
	}

	var err error
	for _, b := range build {
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
