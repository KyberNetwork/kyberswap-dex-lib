package syncswapv2aqua

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	masterABI       abi.ABI
	aquaPoolABI     abi.ABI
	feeManagerV2ABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&masterABI, masterABIData},
		{&aquaPoolABI, aquaPoolABIData},
		{&feeManagerV2ABI, feeManagerV2ABIData},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
