package arenabc

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	tokenManagerABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&tokenManagerABI, tokenManagerBytes},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
