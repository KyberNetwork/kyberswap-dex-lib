package hyperamm

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	hyperAMMFactoryABI       abi.ABI
	hyperAMMABI              abi.ABI
	hyperAMMSwapFeeModuleABI abi.ABI
	hyperAMMLensABI          abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&hyperAMMFactoryABI, hyperAMMFactoryBytes},
		{&hyperAMMABI, hyperAMMBytes},
		{&hyperAMMSwapFeeModuleABI, hyperAMMSwapFeeModuleBytes},
		{&hyperAMMLensABI, hyperAMMLensBytes},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
