package stonkbrokersfunv2

import (
	abiutil "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

var (
	PadABI          = abiutil.MustParseABI(padBytes)
	lensABI         = abiutil.MustParseABI(lensBytes)
	aggregatorV3ABI = abiutil.MustParseABI(aggregatorV3Bytes)
	twapPoolABI     = abiutil.MustParseABI(twapPoolBytes)
)
