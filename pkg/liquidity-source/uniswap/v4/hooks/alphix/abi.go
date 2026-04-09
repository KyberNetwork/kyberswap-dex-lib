package alphix

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	alphixHookABI  abi.ABI
	lvrFeeHookABI  abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&alphixHookABI, alphixHookABIJson},
		{&lvrFeeHookABI, lvrFeeHookABIJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
