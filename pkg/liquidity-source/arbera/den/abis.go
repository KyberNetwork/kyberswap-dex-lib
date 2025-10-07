package arberaden

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	indexManagerABI  abi.ABI
	weightedIndexABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&indexManagerABI, indexManagerABIData},
		{&weightedIndexABI, weightedIndexABIData},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
