package math

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	maxI256 = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 255), big.NewInt(1)) // 2^255 - 1
	minI256 = new(big.Int).Neg(new(big.Int).Lsh(big.NewInt(1), 255))                // -2^255

	bignumber1e19 = bignumber.TenPowInt(19)
)
