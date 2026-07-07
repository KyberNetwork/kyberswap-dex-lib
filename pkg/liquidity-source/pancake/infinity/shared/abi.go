package shared

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"

	bin "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/bin/abi"
	cl "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/cl/abi"

	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"
)

var (
	quoterABI         abi.ABI
	BinPoolManagerABI abi.ABI
	CLPoolManagerABI  abi.ABI
)

var (
	BinPoolManagerFilterer *bin.PancakeInfinityPoolManagerFilterer
	CLPoolManagerFilterer  *cl.PancakeInfinityPoolManagerFilterer
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&quoterABI, quoterABIJson},
		{&BinPoolManagerABI, binPoolManagerABIJson},
		{&CLPoolManagerABI, clPoolManagerABIJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}

	BinPoolManagerFilterer = lo.Must(bin.NewPancakeInfinityPoolManagerFilterer(common.Address{}, nil))
	CLPoolManagerFilterer = lo.Must(cl.NewPancakeInfinityPoolManagerFilterer(common.Address{}, nil))
}
