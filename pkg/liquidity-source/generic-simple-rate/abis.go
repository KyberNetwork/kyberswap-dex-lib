package generic_simple_rate

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	rateABI abi.ABI

	abiMap map[string]abi.ABI
)

func init() {
	abiMap = make(map[string]abi.ABI)

	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&rateABI, rateABIData},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
