package saturnusdat

import (
	abiutil "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

var (
	swapFacilityABI = abiutil.MustParseABI(swapFacilityABIJson)
	usdatABI        = abiutil.MustParseABI(usdatABIJson)
)
