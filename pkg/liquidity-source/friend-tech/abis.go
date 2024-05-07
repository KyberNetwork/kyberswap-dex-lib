package friendtech

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	bunnySwapABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{
			&bunnySwapABI, bunnySwapABIJson,
		},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
