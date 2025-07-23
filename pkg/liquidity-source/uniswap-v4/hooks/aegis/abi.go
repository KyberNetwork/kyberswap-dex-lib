package aegis

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	aegisHookABI              abi.ABI
	aegisDynamicFeeManagerABI abi.ABI
	aegisPoolPolicyManagerABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&aegisHookABI, aegisHookABIJson},
		{&aegisDynamicFeeManagerABI, aegisDynamicFeeManagerABIJson},
		{&aegisPoolPolicyManagerABI, aegisPoolPolicyManagerABIJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
