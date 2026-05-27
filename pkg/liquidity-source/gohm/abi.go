package gohm

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	olympusStakingABI abi.ABI
	gohmABI           abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&olympusStakingABI, olympusStakingABIJson},
		{&gohmABI, gohmABIJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
