package syncswap

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	masterABI      abi.ABI
	classicPoolABI abi.ABI
	stablePoolABI  abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&masterABI, masterABIData},
		{&classicPoolABI, classicPoolABIData},
		{&stablePoolABI, stablePoolABIData},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
