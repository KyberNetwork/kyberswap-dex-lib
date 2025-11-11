package uniswapv4

import (
	"bytes"

	abis "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/abi"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"
)

var (
	stateViewABI   abi.ABI
	poolManagerABI abi.ABI
)

var (
	poolManagerFilterer *abis.UniswapV4PoolManagerFilterer
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&stateViewABI, stateViewABIJson},
		{&poolManagerABI, poolManagerABIJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}

func init() {
	poolManagerFilterer = lo.Must(abis.NewUniswapV4PoolManagerFilterer(common.Address{}, nil))
}
