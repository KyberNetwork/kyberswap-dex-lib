package angletransmuter

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	transmuterABI abi.ABI
	pythABI       abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&transmuterABI, TransmuterJson},
		{&pythABI, PythJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
