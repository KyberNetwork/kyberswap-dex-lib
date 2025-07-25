package v2

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	cTokenABI      abi.ABI
	comptrollerABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&cTokenABI, cTokenJson},
		{&comptrollerABI, comptrollerJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
