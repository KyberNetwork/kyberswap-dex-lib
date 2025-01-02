package beets_ss

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	sonicStakingABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{
			&sonicStakingABI, sonicStakingABIJson,
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
