package abis

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"
)

var (
	DexV2ABI     abi.ABI
	LiquidityABI abi.ABI
	ResolverABI  abi.ABI

	DexV2PoolFilterer *FluidDexV2Filterer
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&DexV2ABI, dexV2Json},
		{&LiquidityABI, liquidityJson},
		{&ResolverABI, resolverJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}

	DexV2PoolFilterer = lo.Must(NewFluidDexV2Filterer(common.Address{}, nil))
}
