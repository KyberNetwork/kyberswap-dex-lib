package pamm

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	lensABI   abi.ABI
	routerABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&lensABI, lensABIData},
		{&routerABI, routerABIData},
	}

	for _, b := range builder {
		parsed, err := abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
		*b.ABI = parsed
	}
}
