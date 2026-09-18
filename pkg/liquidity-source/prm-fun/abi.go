package prmfun

import (
	abiutil "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

var (
	memeCurveABI   = abiutil.MustParseABI(memeCurveABIJson)
	memeFactoryABI = abiutil.MustParseABI(memeFactoryABIJson)
	RouterABI      = abiutil.MustParseABI(premiumRouterABIJson)
)
