package unieth

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	RockXETHABI abi.ABI
	StakingABI  abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{
			&RockXETHABI, rockXETHABIJson,
		},
		{
			&StakingABI, stakingABIJson,
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
