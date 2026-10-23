package upshiftusdf

import (
	abiutil "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

var (
	vaultABI        = abiutil.MustParseABI(vaultABIJson)
	receiptTokenABI = abiutil.MustParseABI(receiptTokenABIJson)
)
