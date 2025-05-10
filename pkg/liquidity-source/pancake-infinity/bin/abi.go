package bin

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var binPoolManagerABI abi.ABI

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&binPoolManagerABI, binPoolManagerABIJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
