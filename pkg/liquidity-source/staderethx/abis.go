package staderethx

import (
	"bytes"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	staderStakePoolsManagerABI abi.ABI
	staderOracleABI            abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&staderStakePoolsManagerABI, staderStakePoolsManagerABIJson},
		{&staderOracleABI, staderOracleABIJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
