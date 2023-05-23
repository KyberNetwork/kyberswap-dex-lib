package polydex

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	pairABI           abi.ABI
	polydexFactoryABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&pairABI, pairJson},
		{&polydexFactoryABI, polydexFactoryJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
