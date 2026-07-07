package limitorder

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/cl"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var _ = cl.RegisterHooksFactory(cl.BaseFactory(valueobject.ExchangePancakeInfinityCLLO),
	common.HexToAddress("0x6AdC560aF85377f9a73d17c658D798c9B39186e8"),
)

type Hook struct{} // for codegen
