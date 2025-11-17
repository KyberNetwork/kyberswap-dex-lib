package ramsesv2

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/ramsesv2/abis"
)

var (
	poolV2ABI    abi.ABI
	poolV3ABI    abi.ABI
	factoryV2ABI abi.ABI
)

var (
	factoryFilterer = lo.Must(abis.NewFactoryFilterer(common.Address{}, nil))
	poolFiltererV2  = lo.Must(abis.NewV2PoolFilterer(common.Address{}, nil))
	poolFiltererV3  = lo.Must(abis.NewV3PoolFilterer(common.Address{}, nil))
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&poolV2ABI, ramsesV2PoolJson},
		{&poolV3ABI, ramsesV3PoolJson},
		{&factoryV2ABI, factoryV2Json},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
