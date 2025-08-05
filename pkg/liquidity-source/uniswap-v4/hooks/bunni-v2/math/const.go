package math

import (
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"

	"github.com/holiman/uint256"
)

var (
	ALPHA_BASE           = uint256.NewInt(1e8)
	SWAP_FEE_BASE        = uint256.NewInt(1e6)
	MODIFIER_BASE        = SWAP_FEE_BASE
	RAW_TOKEN_RATIO_BASE = SWAP_FEE_BASE

	CURATOR_FEE_BASE      = uint256.NewInt(1e5)
	MAX_SWAP_FEE_RATIO, _ = u256.NewUint256("28800000000000000000000") // 2.88e20
	MAX_SWAP_FEE          = SWAP_FEE_BASE                              // 1e6
	MIN_FEE_AMOUNT        = u256.U1
	EPSILON_FEE           = u256.U1
	SWAP_FEE_BASE_SQUARED = uint256.NewInt(1e12)
	LN2_WAD               = uint256.NewInt(693147180559945309)
	WAD                   = uint256.NewInt(1e18)
	Q96                   = new(uint256.Int).Lsh(u256.U1, 96)
)
