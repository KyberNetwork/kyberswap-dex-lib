package math

import (
	"errors"

	"github.com/KyberNetwork/blockchain-toolkit/i256"
	"github.com/KyberNetwork/int256"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"

	"github.com/holiman/uint256"
)

var (
	RESOLUTION            uint = 96
	ALPHA_BASE                 = uint256.NewInt(1e8)
	SWAP_FEE_BASE              = uint256.NewInt(1e6)
	MODIFIER_BASE              = SWAP_FEE_BASE
	RAW_TOKEN_RATIO_BASE       = SWAP_FEE_BASE
	CURATOR_FEE_BASE           = uint256.NewInt(1e5)
	MAX_SWAP_FEE_RATIO, _      = u256.NewUint256("28800000000000000000000") // 2.88e20
	MAX_SWAP_FEE               = SWAP_FEE_BASE                              // 1e6
	MIN_FEE_AMOUNT             = uint256.NewInt(1e3)                        // 1e3
	EPSILON_FEE                = u256.U1
	SWAP_FEE_BASE_SQUARED      = uint256.NewInt(1e12)
	LN2_WAD                    = uint256.NewInt(693147180559945309)
	WAD                        = u256.BONE
	WAD_INT                    = int256.NewInt(1e18)
	Q96                        = new(uint256.Int).Lsh(u256.U1, 96)
	SCALED_Q96, _              = uint256.FromHex("0x10000000000000000000000000")
	_BALANCE_MASK, _           = uint256.FromHex("0x7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")

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

	lnQ96A0 = i256.MustFromDecimal("43456485725739037958740375743393")
	lnQ96A1 = i256.MustFromDecimal("24828157081833163892658089445524")
	lnQ96A2 = i256.MustFromDecimal("3273285459638523848632254066296")
	lnQ96B0 = i256.MustFromDecimal("11111509109440967052023855526967")
	lnQ96B1 = i256.MustFromDecimal("45023709667254063763336534515857")
	lnQ96B2 = i256.MustFromDecimal("14706773417378608786704636184526")
	lnQ96C  = i256.MustFromDecimal("795164235651350426258249787498")

	lnQ96Q0 = i256.MustFromDecimal("5573035233440673466300451813936")
	lnQ96Q1 = i256.MustFromDecimal("71694874799317883764090561454958")
	lnQ96Q2 = i256.MustFromDecimal("283447036172924575727196451306956")
	lnQ96Q3 = i256.MustFromDecimal("401686690394027663651624208769553")
	lnQ96Q4 = i256.MustFromDecimal("204048457590392012362485061816622")
	lnQ96Q5 = i256.MustFromDecimal("31853899698501571402653359427138")
	lnQ96Q6 = i256.MustFromDecimal("909429971244387300277376558375")

	// Scale factor s * 2**96 used after p/q
	lnQ96Scale = i256.MustFromDecimal("439668470185123797622540459591")
	// ln(2) * 2**192 used for adding k * ln(2)
	lnQ96Ln2Scaled2Pow192 = i256.MustFromDecimal("4350955369971217654477563090224794165364344896676135745069")

	ErrOverflow = errors.New("overflow")
)
