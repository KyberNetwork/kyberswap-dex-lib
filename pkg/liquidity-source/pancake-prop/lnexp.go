package pancakeprop

import "github.com/holiman/uint256"

// Solady FixedPointMathLib lnWad/expWad/powWad, ported opcode for opcode. Signed int256 values
// live in uint256 two's complement: Add/Sub/Mul wrap like the EVM, SRsh and SDiv are signed.

func mustU(s string) *uint256.Int { return uint256.MustFromDecimal(s) }

var (
	// lnWad
	lnA1 = mustU("3273285459638523848632254066296")
	lnA2 = mustU("24828157081833163892658089445524")
	lnA3 = mustU("43456485725739037958740375743393")
	lnP0 = mustU("11111509109440967052023855526967")
	lnP1 = mustU("45023709667254063763336534515857")
	lnP2 = mustU("14706773417378608786704636184526")
	lnP3 = new(uint256.Int).Lsh(mustU("795164235651350426258249787498"), 96)
	lnQ  = [...]*uint256.Int{
		mustU("71694874799317883764090561454958"),
		mustU("283447036172924575727196451306956"),
		mustU("401686690394027663651624208769553"),
		mustU("204048457590392012362485061816622"),
		mustU("31853899698501571402653359427138"),
		mustU("909429971244387300277376558375"),
	}
	lnQ0 = mustU("5573035233440673466300451813936")
	lnS  = mustU("1677202110996718588342820967067443963516166")
	lnK  = mustU("16597577552685614221487285958193947469193820559219878177908093499208371")
	lnC  = mustU("600920179829731861736702779321621459595472258049074101567377883020018308")

	// expWad
	expMin   = new(uint256.Int).Neg(mustU("41446531673892822313"))
	expMax   = mustU("135305999368893231589")
	fiveTo18 = mustU("3814697265625")
	ln2Sc    = mustU("54916777467707473351141471128")
	twoTo95  = new(uint256.Int).Lsh(uint256.NewInt(1), 95)
	exC1     = mustU("1346386616545796478920950773328")
	exC2     = mustU("57155421227552351082224309758442")
	exC3     = mustU("94201549194550492254356042504812")
	exC4     = mustU("28719021644029726153956944680412240")
	exC5     = new(uint256.Int).Lsh(mustU("4385272521454847904659076985693276"), 96)
	exD1     = mustU("2855989394907223263936484059900")
	exD2     = mustU("50020603652535783019961831881945")
	exD3     = mustU("533845033583426703283633433725380")
	exD4     = mustU("3604857256930695427073651918091429")
	exD5     = mustU("14423608567350463180887372962807573")
	exD6     = mustU("26449188498355588339934803723976023")
	exS      = mustU("3822833074963236453042738258902158003155416615667")
)

// lnWad returns ln(x/1e18)*1e18 as a two's complement int256, for x > 0.
func lnWad(z, x *uint256.Int) error {
	if x.Sign() <= 0 {
		return ErrLnUndefined
	}
	var r, xn, p, q, t uint256.Int
	r.SetUint64(uint64(256 - x.BitLen())) // 255 ^ log2(x)
	xn.Lsh(x, uint(r.Uint64()))
	xn.Rsh(&xn, 159)

	t.Add(lnA1, &xn).Mul(&t, &xn).SRsh(&t, 96)
	t.Add(lnA2, &t).Mul(&t, &xn).SRsh(&t, 96)
	t.Add(lnA3, &t).Mul(&t, &xn).SRsh(&t, 96)
	p.Sub(&t, lnP0)
	p.Mul(&p, &xn).SRsh(&p, 96)
	p.Sub(&p, lnP1)
	p.Mul(&p, &xn).SRsh(&p, 96)
	p.Sub(&p, lnP2)
	p.Mul(&p, &xn)
	p.Sub(&p, lnP3)

	q.Add(lnQ0, &xn)
	for _, c := range lnQ {
		q.Mul(&xn, &q).SRsh(&q, 96)
		q.Add(c, &q)
	}

	p.SDiv(&p, &q)
	p.Mul(lnS, &p)
	t.SetUint64(159)
	t.Sub(&t, &r).Mul(lnK, &t)
	p.Add(&t, &p)
	p.Add(lnC, &p)
	z.SRsh(&p, 174)
	return nil
}

// expWad returns exp(x/1e18)*1e18 for a two's complement int256 x.
func expWad(z, x *uint256.Int) error {
	if !x.Sgt(expMin) {
		z.Clear()
		return nil
	}
	if !x.Slt(expMax) {
		return ErrExpOverflow
	}
	var v, k, t, y, p, q uint256.Int
	v.Lsh(x, 78)
	v.SDiv(&v, fiveTo18)

	t.Lsh(&v, 96)
	t.SDiv(&t, ln2Sc).Add(&t, twoTo95)
	k.SRsh(&t, 96)
	t.Mul(&k, ln2Sc)
	v.Sub(&v, &t)

	y.Add(&v, exC1)
	y.Mul(&y, &v).SRsh(&y, 96)
	y.Add(&y, exC2)
	p.Add(&y, &v)
	p.Sub(&p, exC3)
	p.Mul(&p, &y).SRsh(&p, 96)
	p.Add(&p, exC4)
	p.Mul(&p, &v)
	p.Add(&p, exC5)

	q.Sub(&v, exD1)
	q.Mul(&q, &v).SRsh(&q, 96)
	q.Add(&q, exD2)
	q.Mul(&q, &v).SRsh(&q, 96)
	q.Sub(&q, exD3)
	q.Mul(&q, &v).SRsh(&q, 96)
	q.Add(&q, exD4)
	q.Mul(&q, &v).SRsh(&q, 96)
	q.Sub(&q, exD5)
	q.Mul(&q, &v).SRsh(&q, 96)
	q.Add(&q, exD6)

	p.SDiv(&p, &q)
	p.Mul(&p, exS)
	// shift = 195 - k with k in [-61, 195]
	z.Rsh(&p, uint(195-int64(k.Uint64())))
	return nil
}

// powWad is expWad(lnWad(x) * y / 1e18) for x > 0 and 0 < y < 2^63.
func powWad(z, x, y *uint256.Int) error {
	var l uint256.Int
	if err := lnWad(&l, x); err != nil {
		return err
	}
	l.Mul(&l, y)
	l.SDiv(&l, uWad)
	return expWad(z, &l)
}
