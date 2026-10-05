package nemoprop

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var nemoSwapABI abi.ABI

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&nemoSwapABI, nemoSwapABIData},
	}

	for _, b := range builder {
		parsed, err := abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
		*b.ABI = parsed
	}
}
