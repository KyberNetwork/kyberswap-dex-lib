package upshiftvaultv2

import (
	abiutil "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

var (
	vaultABI           = abiutil.MustParseABI(vaultABIJson)
	assetsWhitelistABI = abiutil.MustParseABI(assetsWhitelistABIJson)
	erc20ABI           = abiutil.MustParseABI(erc20ABIJson)
)
