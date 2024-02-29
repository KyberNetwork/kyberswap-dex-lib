package susde

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	stakedUSDeV2ABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&stakedUSDeV2ABI, stakedUSDeV2JSON},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
