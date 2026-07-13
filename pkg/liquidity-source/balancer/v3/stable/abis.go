package stable

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	PoolABI        abi.ABI
	StableSurgeABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&PoolABI, poolJson},
		{&StableSurgeABI, stableSurgeJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
