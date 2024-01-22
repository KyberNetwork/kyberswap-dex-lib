package uni

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	UniswapV2Router02ABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&UniswapV2Router02ABI, uniswapV2Router02ABIJSON},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
