package parallelparallelizer

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	parallelizerABI abi.ABI
	chainlinkABI  abi.ABI
	morphoABI     abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&parallelizerABI, ParallelizerJson},
		{&chainlinkABI, ChainlinkJson},
		{&morphoABI, MorphoJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
