package abis

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"
)

var (
	// UniswapV3PoolABI is used both to pack RPC calls (packing only depends on method
	// inputs, identical across all forks) and to unpack the "standard" 7-word slot0()
	// shape shared by uniswap-v3/pancake-v3/ramses-v2/nuri-v2.
	UniswapV3PoolABI abi.ABI
	// Slot0KatanaABI, Slot0SlipstreamABI, and Slot0SolidlyABI unpack the other slot0()
	// shapes seen across uniswap-v3 forks. Try katana (8-word), then standard (7-word),
	// then slipstream (6-word), then solidly (4-word) - longest to shortest to prevent
	// silent misdecode of trailing bytes.
	Slot0KatanaABI     abi.ABI
	Slot0SlipstreamABI abi.ABI
	Slot0SolidlyABI    abi.ABI

	UniswapV3FactoryABI abi.ABI
)

var UniswapV3FactoryFilterer *FactoryFilterer

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&Slot0KatanaABI, slot0KatanaJson},
		{&UniswapV3PoolABI, uniswapV3PoolJson},
		{&Slot0SlipstreamABI, slot0SlipstreamJson},
		{&Slot0SolidlyABI, slot0SolidlyJson},
		{&UniswapV3FactoryABI, uniswapV3FactoryJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}

	UniswapV3FactoryFilterer = lo.Must(NewFactoryFilterer(common.Address{}, nil))
}
