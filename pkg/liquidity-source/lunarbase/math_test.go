package lunarbase

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

func u(s string) *uint256.Int {
	v, _ := uint256.FromDecimal(s)
	return v
}

func assertXToY(t *testing.T, name string,
	pX96 string, fee uint64, resX, resY string, k uint32, dx string,
	expectedDy, expectedPNext, expectedFee string,
) {
	t.Helper()
	params := &PoolParams{
		SqrtPriceX96:   u(pX96),
		FeeQ48:         fee,
		ReserveX:       u(resX),
		ReserveY:       u(resY),
		ConcentrationK: k,
	}
	result := quoteXToY(params, u(dx))
	assert.Equal(t, expectedDy, result.AmountOut.Dec(), "%s: dy mismatch", name)
	assert.Equal(t, expectedPNext, result.SqrtPriceNext.Dec(), "%s: pNext mismatch", name)
	assert.Equal(t, expectedFee, result.Fee.Dec(), "%s: fee mismatch", name)
}

func assertYToX(t *testing.T, name string,
	pX96 string, fee uint64, resX, resY string, k uint32, dy string,
	expectedDx, expectedPNext, expectedFee string,
) {
	t.Helper()
	params := &PoolParams{
		SqrtPriceX96:   u(pX96),
		FeeQ48:         fee,
		ReserveX:       u(resX),
		ReserveY:       u(resY),
		ConcentrationK: k,
	}
	result := quoteYToX(params, u(dy))
	assert.Equal(t, expectedDx, result.AmountOut.Dec(), "%s: dx mismatch", name)
	assert.Equal(t, expectedPNext, result.SqrtPriceNext.Dec(), "%s: pNext mismatch", name)
	assert.Equal(t, expectedFee, result.Fee.Dec(), "%s: fee mismatch", name)
}

func TestVector01_XToY_Price1_Fee5pct_EqualReserves(t *testing.T) {
	assertXToY(t, "V1",
		"79228162514264337593543950336", 14073748835532,
		"1000000000000000000000", "1000000000000000000000", 5000,
		"1000000000000000000",
		"949975824199448327", "79226146299258815947800348471", "49998727589441657",
	)
}

func TestVector02_XToY_SmallSwap(t *testing.T) {
	assertXToY(t, "V2",
		"79228162514264337593543950336", 14073748835532,
		"1000000000000000000000", "1000000000000000000000", 5000,
		"1000000000000000",
		"949999975945444", "79228160508160605167902024956", "49999998733967",
	)
}

func TestVector03_XToY_1pctFee_LargeSwap(t *testing.T) {
	assertXToY(t, "V3",
		"79228162514264337593543950336", 2814749767106,
		"1000000000000000000000", "1000000000000000000000", 5000,
		"10000000000000000000",
		"9899254748004130552", "79222198378280054705185394032", "99992472202041830",
	)
}

func TestVector04_XToY_Price2000_03pctFee(t *testing.T) {
	assertXToY(t, "V4",
		"3543191142285914205922034323968", 844424930131,
		"100000000000000000000", "200000000000000000000000", 5000,
		"1000000000000000000",
		"1993955084591778902382", "3543111330913504216594035255575", "5999864848313399005",
	)
}

func TestVector05_XToY_AsymmetricReserves(t *testing.T) {
	assertXToY(t, "V5",
		"79228162514264337593543950336", 14073748835532,
		"500000000000000000000", "2000000000000000000000", 5000,
		"1000000000000000000",
		"949950918456655526", "79224069208482472228497886508", "49997416760873614",
	)
}

func TestVector06_XToY_K_Zero(t *testing.T) {
	assertXToY(t, "V6",
		"79228162514264337593543950336", 14073748835532,
		"1000000000000000000000", "1000000000000000000000", 0,
		"1000000000000000000",
		"949975946048997793", "79226156461275560551536364690", "49998734002575839",
	)
}

func TestVector07_XToY_LargeK(t *testing.T) {
	assertXToY(t, "V7",
		"79228162514264337593543950336", 14073748835532,
		"1000000000000000000000", "1000000000000000000000", 50000,
		"1000000000000000000",
		"949974726932074358", "79226054789282156509522831410", "49998669838527237",
	)
}

func TestVector08_XToY_TinyFee(t *testing.T) {
	assertXToY(t, "V8",
		"79228162514264337593543950336", 28147497671,
		"1000000000000000000000", "1000000000000000000000", 5000,
		"1000000000000000000",
		"999899949698713205", "79228158528587226556312313483", "99999994969135",
	)
}

func TestVector09_XToY_PriceHalf(t *testing.T) {
	assertXToY(t, "V9",
		"56022770974786139918731938227", 14073748835532,
		"1000000000000000000000", "1000000000000000000000", 5000,
		"1000000000000000000",
		"474987912099724164", "56021345295483405490666417659", "24999363794720828",
	)
}

func TestVector10_XToY_LargeRelativeSwap(t *testing.T) {
	assertXToY(t, "V10",
		"79228162514264337593543950336", 14073748835532,
		"100000000000000000000", "100000000000000000000", 5000,
		"5000000000000000000",
		"4650044428553448897", "77560942249259155893528060534", "244739180450166876",
	)
}

func TestVector11_YToX_Price1_Fee5pct(t *testing.T) {
	assertYToX(t, "V11",
		"79228162514264337593543950336", 14073748835532,
		"1000000000000000000000", "1000000000000000000000", 5000,
		"1000000000000000000",
		"949975824199448327", "79230178780580230100134035213", "49998727589441657",
	)
}

func TestVector12_YToX_Price2000(t *testing.T) {
	assertYToX(t, "V12",
		"3543191142285914205922034323968", 844424930131,
		"100000000000000000000", "200000000000000000000000", 5000,
		"2000000000000000000000",
		"996977542295889451", "3543270955456138198549696866767", "2999932424156699",
	)
}

func TestVector13_YToX_Asymmetric(t *testing.T) {
	assertYToX(t, "V13",
		"79228162514264337593543950336", 14073748835532,
		"2000000000000000000000", "500000000000000000000", 5000,
		"1000000000000000000",
		"949950918456655526", "79232256031536882385322409757", "49997416760873614",
	)
}

func TestVector14_YToX_SmallAmount(t *testing.T) {
	assertYToX(t, "V14",
		"79228162514264337593543950336", 2814749767106,
		"1000000000000000000000", "1000000000000000000000", 5000,
		"1000000000000000",
		"989999995037556", "79228162911401191703432503872", "9999999949872",
	)
}

func TestVector15_YToX_PriceHalf(t *testing.T) {
	assertYToX(t, "V15",
		"56022770974786139918731938227", 14073748835532,
		"1000000000000000000000", "1000000000000000000000", 5000,
		"1000000000000000000",
		"1899903299258648504", "56025622405955431136665335759", "99994910487291306",
	)
}

func TestIsqrt(t *testing.T) {
	cases := []struct {
		input, expected uint64
	}{
		{0, 0},
		{1, 1},
		{4, 2},
		{9, 3},
		{10, 3},
		{100, 10},
	}
	for _, tc := range cases {
		got := isqrt(uint256.NewInt(tc.input))
		assert.Equal(t, uint256.NewInt(tc.expected), got, "isqrt(%d)", tc.input)
	}
}

func TestConcentrationQ48_ZeroFee(t *testing.T) {
	c := concentrationQ48(0, uint256.NewInt(1000), uint256.NewInt(10000), 5000)
	assert.True(t, c.IsZero())
}

func TestConcentrationQ48_ZeroAmount(t *testing.T) {
	c := concentrationQ48(1000, new(uint256.Int), uint256.NewInt(10000), 5000)
	assert.Equal(t, uint256.NewInt(1000), c)
}

func TestQuoteReturnsZeroWhenNoLiquidity(t *testing.T) {
	params := &PoolParams{
		SqrtPriceX96:   new(uint256.Int).Lsh(uint256.NewInt(1), 96),
		FeeQ48:         1 << 44,
		ReserveX:       new(uint256.Int),
		ReserveY:       new(uint256.Int),
		ConcentrationK: 5000,
	}
	result := quoteXToY(params, uint256.NewInt(1000))
	assert.True(t, result.AmountOut.IsZero())
}
