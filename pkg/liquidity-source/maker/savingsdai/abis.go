package savingsdai

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	potABI        abi.ABI
	savingsdaiABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&potABI, potJSON},
		{&savingsdaiABI, savingsdaiJSON},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
