package math

import (
	"errors"

	"github.com/KyberNetwork/blockchain-toolkit/i256"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"

	"github.com/holiman/uint256"
)

var (
	ALPHA_BASE            = uint256.NewInt(1e8)
	SWAP_FEE_BASE         = uint256.NewInt(1e6)
	MODIFIER_BASE         = SWAP_FEE_BASE
	RAW_TOKEN_RATIO_BASE  = SWAP_FEE_BASE
	CURATOR_FEE_BASE      = uint256.NewInt(1e5)
	MAX_SWAP_FEE_RATIO, _ = u256.NewUint256("28800000000000000000000") // 2.88e20
	MAX_SWAP_FEE          = SWAP_FEE_BASE                              // 1e6
	MIN_FEE_AMOUNT        = u256.U1
	EPSILON_FEE           = u256.U1
	SWAP_FEE_BASE_SQUARED = uint256.NewInt(1e12)
	LN2_WAD               = uint256.NewInt(693147180559945309)
	WAD                   = u256.BONE
	Q96                   = new(uint256.Int).Lsh(u256.U1, 96)

	minX        = i256.MustFromDecimal("-42139678854452767551")
	maxX        = i256.MustFromDecimal("135305999368893231589")
	fiveToThe18 = i256.MustFromDecimal("3814697265625")
	ln2Scaled   = i256.MustFromDecimal("54916777467707473351141471128")
	twoTo95     = i256.MustFromDecimal("39614081257132168796771975168")

	p0 = i256.MustFromDecimal("1346386616545796478920950773328")
	p1 = i256.MustFromDecimal("57155421227552351082224309758442")
	p2 = i256.MustFromDecimal("94201549194550492254356042504812")
	p3 = i256.MustFromDecimal("28719021644029726153956944680412240")
	p4 = i256.MustFromDecimal("4385272521454847904659076985693276")

	q0 = i256.MustFromDecimal("2855989394907223263936484059900")
	q1 = i256.MustFromDecimal("50020603652535783019961831881945")
	q2 = i256.MustFromDecimal("533845033583426703283633433725380")
	q3 = i256.MustFromDecimal("3604857256930695427073651918091429")
	q4 = i256.MustFromDecimal("14423608567350463180887372962807573")
	q5 = i256.MustFromDecimal("26449188498355588339934803723976023")

	scaleFactor = uint256.MustFromDecimal("3822833074963236453042738258902158003155416615667")

	ErrOverflow = errors.New("overflow")
)
