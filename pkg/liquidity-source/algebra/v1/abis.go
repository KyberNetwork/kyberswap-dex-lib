package algebrav1

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/algebra/v1/abis"
)

var (
	algebraV1FactoryABI                   abi.ABI
	algebraV1PoolABI                      abi.ABI
	algebraV1DirFeePoolABI                abi.ABI
	algebraV1DataStorageOperatorABI       abi.ABI
	algebraV1DirFeeDataStorageOperatorABI abi.ABI
	ticklensABI                           abi.ABI
)

var (
	poolFilterer    = lo.Must(abis.NewPoolFilterer(common.Address{}, nil))
	factoryFilterer = lo.Must(abis.NewFactoryFilterer(common.Address{}, nil))
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&algebraV1FactoryABI, algebraV1FactoryJson},
		{&algebraV1PoolABI, algebraV1PoolJson},
		{&algebraV1DirFeePoolABI, algebraV1DirFeePoolJson},
		{&algebraV1DataStorageOperatorABI, algebraV1DataStorageOperatorJson},
		{&algebraV1DirFeeDataStorageOperatorABI, algebraV1DirFeeDataStorageOperatorJson},
		{&ticklensABI, ticklensJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
