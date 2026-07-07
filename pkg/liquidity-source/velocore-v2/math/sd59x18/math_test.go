package sd59x18

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLog2(t *testing.T) {
	t.Parallel()
	t.Run("1. should return correct value", func(t *testing.T) {
		expected := "10275233743784123062"

		x, _ := new(big.Int).SetString("1239234710472810957214", 10)
		res, err := new(SD59x18).Log2(SD(x))
		assert.Nil(t, err)

		assert.Equal(t, expected, res.value.String())
	})

	t.Run("2. should return correct value", func(t *testing.T) {
		expected := "90825107446694031641"

		x, _ := new(big.Int).SetString("2193217491592174921591742914732915624927147914", 10)
		res, err := new(SD59x18).Log2(SD(x))
		assert.Nil(t, err)

		assert.Equal(t, expected, res.value.String())
	})

	t.Run("3. should return correct value", func(t *testing.T) {
		expected := "-59794705707972522245"

		x, _ := new(big.Int).SetString("1", 10)
		res, err := new(SD59x18).Log2(SD(x))
		assert.Nil(t, err)

		assert.Equal(t, expected, res.value.String())
	})

	t.Run("4. should return error", func(t *testing.T) {
		x, _ := new(big.Int).SetString("0", 10)
		_, err := new(SD59x18).Log2(SD(x))
		assert.ErrorIs(t, err, Err_PRBMath_SD59x18_Log_InputTooSmall)
	})
}

func TestExp2(t *testing.T) {
	t.Parallel()
	t.Run("1. should return correct value", func(t *testing.T) {
		expected := "991501979360783687"

		x, _ := new(big.Int).SetString("-12312442341321497", 10)
		res, err := new(SD59x18).Exp2(SD(x))
		assert.Nil(t, err)

		assert.Equal(t, expected, res.value.String())
	})

	t.Run("2. should return correct value", func(t *testing.T) {
		expected := "1000009852095982467"

		x, _ := new(big.Int).SetString("14213500000000", 10)
		res, err := new(SD59x18).Exp2(SD(x))
		assert.Nil(t, err)

		assert.Equal(t, expected, res.value.String())
	})

	t.Run("3. should return correct value", func(t *testing.T) {
		expected := "745770835772355297"

		x, _ := new(big.Int).SetString("-423195714924214000", 10)
		res, err := new(SD59x18).Exp2(SD(x))
		assert.Nil(t, err)

		assert.Equal(t, expected, res.value.String())
	})

	t.Run("4. should return error", func(t *testing.T) {
		x, _ := new(big.Int).SetString("20439174392159174914243", 10)
		_, err := new(SD59x18).Exp2(SD(x))
		assert.ErrorIs(t, err, Err_PRBMath_SD59x18_Exp2_InputTooBig)
	})
}

func TestPow(t *testing.T) {
	t.Parallel()
	t.Run("1. should return correct value", func(t *testing.T) {
		expected := "1000000007819231118"

		x, _ := new(big.Int).SetString("13894732914", 10)
		y, _ := new(big.Int).SetString("-432198571", 10)

		res, err := new(SD59x18).Pow(SD(x), SD(y))
		assert.Nil(t, err)

		assert.Equal(t, expected, res.value.String())
	})

	t.Run("2. should return correct value", func(t *testing.T) {
		expected := "999999999999999960"

		x, _ := new(big.Int).SetString("1", 10)
		y, _ := new(big.Int).SetString("1", 10)

		res, err := new(SD59x18).Pow(SD(x), SD(y))
		assert.Nil(t, err)

		assert.Equal(t, expected, res.value.String())
	})

	t.Run("3. should return correct value", func(t *testing.T) {
		expected := "999999999999995856"

		x, _ := new(big.Int).SetString("1", 10)
		y, _ := new(big.Int).SetString("100", 10)

		res, err := new(SD59x18).Pow(SD(x), SD(y))
		assert.Nil(t, err)

		assert.Equal(t, expected, res.value.String())
	})

	t.Run("4. should return error", func(t *testing.T) {
		x, _ := new(big.Int).SetString("-1", 10)
		y, _ := new(big.Int).SetString("100", 10)

		_, err := new(SD59x18).Pow(SD(x), SD(y))
		assert.ErrorIs(t, err, Err_PRBMath_SD59x18_Log_InputTooSmall)
	})
}

func TestMul(t *testing.T) {
	t.Parallel()
	t.Run("1. should return correct value", func(t *testing.T) {
		expected := "0"

		x, _ := new(big.Int).SetString("-31471942", 10)
		y, _ := new(big.Int).SetString("432150392", 10)

		res, err := new(SD59x18).Mul(SD(x), SD(y))
		assert.Nil(t, err)

		assert.Equal(t, expected, res.value.String())
	})

	t.Run("2. should return correct value", func(t *testing.T) {
		expected := "4576679332325557870713639756650141931978623606409093849222065"

		x, _ := new(big.Int).SetString("2139317492174912748921478291478129471294", 10)
		y, _ := new(big.Int).SetString("2139317492174912748921478291478129471294", 10)

		res, err := new(SD59x18).Mul(SD(x), SD(y))
		assert.Nil(t, err)

		assert.Equal(t, expected, res.value.String())
	})

	t.Run("3. should return correct value", func(t *testing.T) {
		expected := "-134833209231304"

		x, _ := new(big.Int).SetString("-431421947249147", 10)
		y, _ := new(big.Int).SetString("312532104801424214", 10)

		res, err := new(SD59x18).Mul(SD(x), SD(y))
		assert.Nil(t, err)

		assert.Equal(t, expected, res.value.String())
	})

	t.Run("4. should return error", func(t *testing.T) {
		x, _ := new(big.Int).SetString("-432108420142414321048230148230153214214", 10)
		y, _ := new(big.Int).SetString("213248104820152271042104810482107310532810482014812304214", 10)

		_, err := new(SD59x18).Mul(SD(x), SD(y))
		assert.ErrorIs(t, err, Err_PRBMath_SD59x18_Mul_Overflow)
	})
}

func TestDiv(t *testing.T) {
	t.Parallel()
	t.Run("1. should return correct value", func(t *testing.T) {
		expected := "29044803764687059979539950758763"

		x, _ := new(big.Int).SetString("412942439471924712956329174921471924234", 10)
		y, _ := new(big.Int).SetString("14217429142144314829147241", 10)

		res, err := new(SD59x18).Div(SD(x), SD(y))
		assert.Nil(t, err)

		assert.Equal(t, expected, res.value.String())
	})

	t.Run("2. should return correct value", func(t *testing.T) {
		expected := "-25258852003234736335344272844272844"

		x, _ := new(big.Int).SetString("-314018048104214242121", 10)
		y, _ := new(big.Int).SetString("12432", 10)

		res, err := new(SD59x18).Div(SD(x), SD(y))
		assert.Nil(t, err)

		assert.Equal(t, expected, res.value.String())
	})

	t.Run("3. should return correct value", func(t *testing.T) {
		expected := "-3476000249353172"

		x, _ := new(big.Int).SetString("-432147293175239147293719471294", 10)
		y, _ := new(big.Int).SetString("124323147921423429372921472394124", 10)

		res, err := new(SD59x18).Div(SD(x), SD(y))
		assert.Nil(t, err)

		assert.Equal(t, expected, res.value.String())
	})

	t.Run("4. should return error", func(t *testing.T) {
		x, _ := new(big.Int).SetString("-432147293175239147293719471294", 10)
		y, _ := new(big.Int).SetString("0", 10)
		_, err := new(SD59x18).Div(SD(x), SD(y))
		assert.ErrorIs(t, err, Err_PRBMath_SD59x18_Div_Overflow)
	})
}
