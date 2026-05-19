package fermi

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	fermiSwapperABI abi.ABI
	fermiEngineABI  abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&fermiSwapperABI, fermiSwapperABIJson},
		{&fermiEngineABI, fermiEngineABIJson},
	}
	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
