package ironstable

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	ironSwap abi.ABI
	erc20    abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&ironSwap, ironSwapBytes},
		{&erc20, erc20Bytes},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
