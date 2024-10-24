package meth

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	mantleLSPStakingABI abi.ABI
	mantlePauserABI     abi.ABI
	methABI             abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&mantleLSPStakingABI, stakingABIJSON},
		{&mantlePauserABI, pauserABIJSON},
		{&methABI, methABIJSON},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
