package maverickv1

import (
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
)

var (
	DefaultGas = Gas{Swap: 125000}
	zeroBI     = big.NewInt(0)
)

var (
	MaxTick                     = 460540
	MaxSwapIterationCalculation = 50
	OffsetMask                  = big.NewInt(255)
	Kinds                       = big.NewInt(4)
	Mask                        = big.NewInt(15)
	BitMask                     = new(big.Int).Sub(new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil), big.NewInt(1))
	WordSize                    = big.NewInt(256)
	One                         = constant.TenPowInt(18)
	Unit                        = constant.TenPowInt(18)
)
