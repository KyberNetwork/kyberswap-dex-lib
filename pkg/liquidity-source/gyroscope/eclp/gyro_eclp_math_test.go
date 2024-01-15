package gyroeclp

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/int256"
	"github.com/stretchr/testify/assert"
)

func Test_calcAChiAChiInXp(t *testing.T) {
	t.Run("1. should return correct result", func(t *testing.T) {
		var (
			alpha  = int256.MustFromDec("2153421421514125231432")
			beta   = int256.MustFromDec("351242142152144352142315")
			c      = int256.MustFromDec("4542142342121424214")
			s      = int256.MustFromDec("552142314423142324214")
			lambda = int256.MustFromDec("532142342142142144365")

			tauAlphaX = int256.MustFromDec("152142154332623423")
			tauAlphaY = int256.MustFromDec("55353432140214225325")
			tauBetaX  = int256.MustFromDec("215432632543253245215")
			tauBetaY  = int256.MustFromDec("6421521532542534246431")
			u         = int256.MustFromDec("1153452543263253253524")
			v         = int256.MustFromDec("326533539257394275394645")
			w         = int256.MustFromDec("269304283058430853402583532")
			z         = int256.MustFromDec("64430286340583402583905215")
			dSq       = int256.MustFromDec("20000000000000000000000000000000000000000")

			// [2153421421514125231432, 351242142152144352142315, 4542142342121424214, 552142314423142324214, 532142342142142144365]
			// [[152142154332623423, 55353432140214225325], [215432632543253245215, 6421521532542534246431], 1153452543263253253524, 326533539257394275394645, 269304283058430853402583532, 64430286340583402583905215, 20000000000000000000000000000000000000000]

			p = &params{
				Alpha:  alpha,
				Beta:   beta,
				C:      c,
				S:      s,
				Lambda: lambda,
			}

			d = &derivedParams{
				TauAlpha: &vector2{
					X: tauAlphaX,
					Y: tauAlphaY,
				},
				TauBeta: &vector2{
					X: tauBetaX,
					Y: tauBetaY,
				},
				U:   u,
				V:   v,
				W:   w,
				Z:   z,
				DSq: dSq,
			}
		)

		expected, _ := new(big.Int).SetString("7519295519963", 10)

		actual, err := GyroECLPMath.calcAChiAChiInXp(p, d)
		assert.Nil(t, err)

		assert.Equal(t, expected.String(), actual.Dec())
	})

	t.Run("2. should return correct result", func(t *testing.T) {

	})

	t.Run("3. should return correct result", func(t *testing.T) {

	})

	t.Run("4. should return correct result", func(t *testing.T) {

	})

	t.Run("5. should return correct result", func(t *testing.T) {

	})

	t.Run("6. should return error", func(t *testing.T) {

	})
}

func Test_calcAtAChi(t *testing.T) {

}

func Test_virtualOffset0(t *testing.T) {

}

func Test_virtualOffset1(t *testing.T) {

}

func Test_maxBalances0(t *testing.T) {

}

func Test_maxBalances1(t *testing.T) {

}

func Test_calcMinAtxAChiySqPlusAtxSq(t *testing.T) {

}

func Test_calc2AtxAtyAChixAChiy(t *testing.T) {

}

func Test_calcMinAtyAChixSqPlusAtySq(t *testing.T) {

}

func Test_calcInvariantSqrt(t *testing.T) {

}

func Test_checkAssetBounds(t *testing.T) {

}

func Test_calcXpXpDivLambdaLambda(t *testing.T) {

}

func Test_solveQuadraticSwap(t *testing.T) {

}

func Test_calcYGivenX(t *testing.T) {

}

func Test_calcXGivenY(t *testing.T) {

}
