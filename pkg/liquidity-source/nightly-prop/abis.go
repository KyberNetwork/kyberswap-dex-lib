package nightlyprop

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	routerABI abi.ABI
	poolABI   abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&routerABI, routerABIData},
		{&poolABI, poolABIData},
	}

	for _, b := range builder {
		parsed, err := abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
		*b.ABI = parsed
	}
}
