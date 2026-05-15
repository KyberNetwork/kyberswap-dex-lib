package nadswap

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	factoryABI      abi.ABI
	pairABI         abi.ABI
	feeCollectorABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&factoryABI, factoryJson},
		{&pairABI, pairJson},
		{&feeCollectorABI, feeCollectorJson},
	}
	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
