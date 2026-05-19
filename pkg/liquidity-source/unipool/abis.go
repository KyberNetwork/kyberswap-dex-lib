package unipool

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	uniPoolFactoryABI abi.ABI
	uniPoolPairABI    abi.ABI
)

func init() {
	for _, b := range []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&uniPoolFactoryABI, factoryABIJson},
		{&uniPoolPairABI, pairABIJson},
	} {
		parsed, err := abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
		*b.ABI = parsed
	}
}
