package integral

import (
	"bytes"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/algebra/integral/abis"
	intergralpoolv10 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/algebra/integral/abis/v10"
	intergralpoolv12 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/algebra/integral/abis/v12"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"
)

var (
	factoryABI      abi.ABI
	poolV10ABI      abi.ABI
	poolV12ABI      abi.ABI
	basePluginV2ABI abi.ABI
	ticklensABI     abi.ABI
)

var (
	factoryFilterer = lo.Must(abis.NewFactoryFilterer(common.Address{}, nil))
	poolV10Filterer = lo.Must(intergralpoolv10.NewPoolFilterer(common.Address{}, nil))
	poolV12Filterer = lo.Must(intergralpoolv12.NewPoolFilterer(common.Address{}, nil))
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&factoryABI, algebraFactoryJson},
		{&poolV10ABI, poolV10Json},
		{&poolV12ABI, poolV12Json},
		{&basePluginV2ABI, basePluginV2Json},
		{&ticklensABI, ticklenJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
