package doppler

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	hookABI              abi.ABI
	poolStateABI         abi.ABI
	rehypeDopplerHookABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&hookABI, hookABIJson},
		{&poolStateABI, poolStateABIJson},
		{&rehypeDopplerHookABI, rehypeDopplerHookABIJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
