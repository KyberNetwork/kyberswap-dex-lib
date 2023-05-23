package camelot

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	camelotFactoryABI abi.ABI
	camelotPairABI    abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&camelotFactoryABI, camelotFactoryBytes},
		{&camelotPairABI, camelotPairBytes},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
