package usd0pp

import (
	"bytes"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	usd0ppABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&usd0ppABI, usd0ppABIJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
