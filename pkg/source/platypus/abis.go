package platypus

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	poolABI       abi.ABI
	assetABI      abi.ABI
	stakedAvaxABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{
			ABI:  &poolABI,
			data: poolABIData,
		},
		{
			ABI:  &assetABI,
			data: assetABIData,
		},
		{
			ABI:  &stakedAvaxABI,
			data: stakedAvaxABIData,
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
