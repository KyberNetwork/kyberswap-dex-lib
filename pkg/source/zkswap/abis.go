package zkswap

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	zkswapPairABI    abi.ABI
	zkswpFactoryABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{
			&zkswapPairABI, pairABIJson,
		},
		{
			&zkswpFactoryABI, factoryABIJson,
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
