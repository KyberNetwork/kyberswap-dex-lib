package baseline

import (
	"errors"
	"math/big"

	"github.com/holiman/uint256"
)

var (
	wadBI       = big.NewInt(1e18)
	twoBI       = big.NewInt(2)
	maxPowArgBI = new(big.Int).Mul(big.NewInt(135), wadBI)

	errTradeExceedsLimit = errors.New("trade exceeds limit")
	errPriceMustChange   = errors.New("price must change")
	errInvalidCurveState = errors.New("invalid curve state")
	errSolverFailed      = errors.New("solver failed")
)

func uToBI(x *uint256.Int) *big.Int {
	if x == nil {
		return new(big.Int)
	}
	return x.ToBig()
}

func biToU(x *big.Int) *uint256.Int {
	if x == nil || x.Sign() <= 0 {
		return uint256.NewInt(0)
	}
	return uint256.MustFromBig(x)
}

func addBI(x, y *big.Int) *big.Int {
	return new(big.Int).Add(x, y)
}

func subBI(x, y *big.Int) *big.Int {
	return new(big.Int).Sub(x, y)
}

func mulBI(x, y *big.Int) *big.Int {
	return new(big.Int).Mul(x, y)
}

func divBI(x, y *big.Int) *big.Int {
	if y.Sign() == 0 {
		return nil
	}
	return new(big.Int).Div(x, y)
}

func ceilDivBI(x, y *big.Int) *big.Int {
	if y.Sign() == 0 {
		return nil
	}
	q, r := new(big.Int).QuoRem(x, y, new(big.Int))
	if r.Sign() > 0 {
		q.Add(q, big.NewInt(1))
	}
	return q
}

func zeroFloorSubBI(x, y *big.Int) *big.Int {
	if x.Cmp(y) <= 0 {
		return new(big.Int)
	}
	return subBI(x, y)
}

func absBI(x *big.Int) *big.Int {
	return new(big.Int).Abs(x)
}

func mulWad(x, y *big.Int) *big.Int {
	return divBI(mulBI(x, y), wadBI)
}

func mulWadUp(x, y *big.Int) *big.Int {
	return ceilDivBI(mulBI(x, y), wadBI)
}

func divWad(x, y *big.Int) *big.Int {
	return divBI(mulBI(x, wadBI), y)
}

func divWadUp(x, y *big.Int) *big.Int {
	return ceilDivBI(mulBI(x, wadBI), y)
}

func fullMulDiv(x, y, d *big.Int) *big.Int {
	return divBI(mulBI(x, y), d)
}

func fullMulDivUp(x, y, d *big.Int) *big.Int {
	return ceilDivBI(mulBI(x, y), d)
}

func normalizeWadBI(amount *big.Int, decimals uint8) *big.Int {
	if decimals < 18 {
		return new(big.Int).Mul(amount, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(18-decimals)), nil))
	}
	if decimals > 18 {
		return new(big.Int).Div(amount, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals-18)), nil))
	}
	return new(big.Int).Set(amount)
}

func denormalizeWadBI(amount *big.Int, decimals uint8) *big.Int {
	if decimals < 18 {
		return new(big.Int).Div(amount, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(18-decimals)), nil))
	}
	if decimals > 18 {
		return new(big.Int).Mul(amount, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals-18)), nil))
	}
	return new(big.Int).Set(amount)
}

func denormalizeWadUpBI(amount *big.Int, decimals uint8) *big.Int {
	if decimals < 18 {
		divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(18-decimals)), nil)
		return ceilDivBI(amount, divisor)
	}
	if decimals > 18 {
		return new(big.Int).Mul(amount, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals-18)), nil))
	}
	return new(big.Int).Set(amount)
}

func toWadSignedBI(amount *big.Int, decimals uint8) *big.Int {
	if amount.Sign() >= 0 {
		return normalizeWadBI(amount, decimals)
	}
	return new(big.Int).Neg(normalizeWadBI(absBI(amount), decimals))
}

func powWadBI(x, y *big.Int) (*big.Int, error) {
	if x.Sign() <= 0 {
		return nil, errInvalidCurveState
	}

	lnX, err := lnWadBI(x)
	if err != nil {
		return nil, err
	}
	exponent := sdivBI(mulBI(lnX, y), wadBI)
	res, err := expWadBI(exponent)
	if err != nil {
		return nil, err
	}
	if res.Sign() == 0 {
		return nil, errTradeExceedsLimit
	}
	return res, nil
}

func lnWadBI(x *big.Int) (*big.Int, error) {
	if x.Sign() <= 0 {
		return nil, errInvalidCurveState
	}
	if x.BitLen() > 256 {
		return nil, errInvalidCurveState
	}

	r := int64(256 - x.BitLen())
	x96 := new(big.Int).Lsh(new(big.Int).Set(x), uint(r))
	x96.Rsh(x96, 159)

	p := addBI(mustBI("43456485725739037958740375743393"), sarBI(mulBI(addBI(mustBI("24828157081833163892658089445524"), sarBI(mulBI(addBI(mustBI("3273285459638523848632254066296"), x96), x96), 96)), x96), 96))
	p = subBI(sarBI(mulBI(p, x96), 96), mustBI("11111509109440967052023855526967"))
	p = subBI(sarBI(mulBI(p, x96), 96), mustBI("45023709667254063763336534515857"))
	p = subBI(sarBI(mulBI(p, x96), 96), mustBI("14706773417378608786704636184526"))
	p = subBI(mulBI(p, x96), new(big.Int).Lsh(mustBI("795164235651350426258249787498"), 96))

	q := addBI(mustBI("5573035233440673466300451813936"), x96)
	q = addBI(mustBI("71694874799317883764090561454958"), sarBI(mulBI(x96, q), 96))
	q = addBI(mustBI("283447036172924575727196451306956"), sarBI(mulBI(x96, q), 96))
	q = addBI(mustBI("401686690394027663651624208769553"), sarBI(mulBI(x96, q), 96))
	q = addBI(mustBI("204048457590392012362485061816622"), sarBI(mulBI(x96, q), 96))
	q = addBI(mustBI("31853899698501571402653359427138"), sarBI(mulBI(x96, q), 96))
	q = addBI(mustBI("909429971244387300277376558375"), sarBI(mulBI(x96, q), 96))

	p = sdivBI(p, q)
	p = mulBI(mustBI("1677202110996718588342820967067443963516166"), p)
	p = addBI(mulBI(mustBI("16597577552685614221487285958193947469193820559219878177908093499208371"), big.NewInt(159-r)), p)
	p = addBI(mustBI("600920179829731861736702779321621459595472258049074101567377883020018308"), p)
	return sarBI(p, 174), nil
}

func checkPowLimit(ratio, convexityExp *big.Int) error {
	if ratio.Cmp(wadBI) == 0 {
		return nil
	}
	lnRatio, err := lnWadBI(ratio)
	if err != nil {
		return err
	}
	if mulWad(convexityExp, lnRatio).Cmp(maxPowArgBI) > 0 {
		return errTradeExceedsLimit
	}
	return nil
}

func expWadBI(x *big.Int) (*big.Int, error) {
	if x.Cmp(mustBI("-41446531673892822313")) <= 0 {
		return new(big.Int), nil
	}
	if x.Cmp(mustBI("135305999368893231589")) >= 0 {
		return nil, errTradeExceedsLimit
	}

	x2 := sdivBI(new(big.Int).Lsh(new(big.Int).Set(x), 78), mustBI("3814697265625"))
	k := sarBI(addBI(sdivBI(new(big.Int).Lsh(new(big.Int).Set(x2), 96), mustBI("54916777467707473351141471128")), new(big.Int).Lsh(big.NewInt(1), 95)), 96)
	x2 = subBI(x2, mulBI(k, mustBI("54916777467707473351141471128")))

	y := addBI(x2, mustBI("1346386616545796478920950773328"))
	y = addBI(sarBI(mulBI(y, x2), 96), mustBI("57155421227552351082224309758442"))
	p := subBI(addBI(y, x2), mustBI("94201549194550492254356042504812"))
	p = addBI(sarBI(mulBI(p, y), 96), mustBI("28719021644029726153956944680412240"))
	p = addBI(mulBI(p, x2), new(big.Int).Lsh(mustBI("4385272521454847904659076985693276"), 96))

	q := subBI(x2, mustBI("2855989394907223263936484059900"))
	q = addBI(sarBI(mulBI(q, x2), 96), mustBI("50020603652535783019961831881945"))
	q = subBI(sarBI(mulBI(q, x2), 96), mustBI("533845033583426703283633433725380"))
	q = addBI(sarBI(mulBI(q, x2), 96), mustBI("3604857256930695427073651918091429"))
	q = subBI(sarBI(mulBI(q, x2), 96), mustBI("14423608567350463180887372962807573"))
	q = addBI(sarBI(mulBI(q, x2), 96), mustBI("26449188498355588339934803723976023"))

	r := sdivBI(p, q)
	r.Mul(r, mustBI("3822833074963236453042738258902158003155416615667"))
	shift := 195 - k.Int64()
	if shift < 0 {
		return nil, errTradeExceedsLimit
	}
	return r.Rsh(r, uint(shift)), nil
}

func sdivBI(x, y *big.Int) *big.Int {
	return new(big.Int).Quo(x, y)
}

func sarBI(x *big.Int, shift uint) *big.Int {
	return new(big.Int).Rsh(x, shift)
}

func mustBI(s string) *big.Int {
	x, ok := new(big.Int).SetString(s, 10)
	if !ok {
		panic("invalid big.Int constant")
	}
	return x
}
