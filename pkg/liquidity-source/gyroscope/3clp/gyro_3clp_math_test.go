package gyro3clp

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/gyroscope/math"
)

func TestGyro3CLPMath_calculateInvariant(t *testing.T) {
	t.Run("1. should return correct result", func(t *testing.T) {
		balances := []*uint256.Int{
			uint256.MustFromDecimal("2140189504379402815"),
			uint256.MustFromDecimal("414832901582314"),
			uint256.MustFromDecimal("512920182014252312"),
		}
		root3Alpha := uint256.MustFromDecimal("812058201482")
		expected := uint256.MustFromDecimal("33602146275841914091")
		actual, err := Gyro3CLPMath._calculateInvariant(balances, root3Alpha)
		assert.Nil(t, err)
		assert.Equal(t, expected.Dec(), actual.Dec())
	})

	t.Run("2. should return correct result", func(t *testing.T) {
		balances := []*uint256.Int{
			uint256.MustFromDecimal("4123483012843210430512"),
			uint256.MustFromDecimal("104328132142150214"),
			uint256.MustFromDecimal("51202808012843221401"),
		}
		root3Alpha := uint256.MustFromDecimal("4312957104")
		expected := uint256.MustFromDecimal("1589443311748584037211695")
		actual, err := Gyro3CLPMath._calculateInvariant(balances, root3Alpha)
		assert.Nil(t, err)
		assert.Equal(t, expected.Dec(), actual.Dec())
	})

	t.Run("3. should return correct result", func(t *testing.T) {
		balances := []*uint256.Int{
			uint256.MustFromDecimal("4432163210430512"),
			uint256.MustFromDecimal("134127042142150214"),
			uint256.MustFromDecimal("512421402808012801"),
		}
		root3Alpha := uint256.MustFromDecimal("15321434214129571")
		expected := uint256.MustFromDecimal("72763504628587906")
		actual, err := Gyro3CLPMath._calculateInvariant(balances, root3Alpha)
		assert.Nil(t, err)
		assert.Equal(t, expected.Dec(), actual.Dec())
	})

	t.Run("4. should return correct result", func(t *testing.T) {
		balances := []*uint256.Int{
			uint256.MustFromDecimal("4432163210430512"),
			uint256.MustFromDecimal("134127042142150214"),
			uint256.MustFromDecimal("512421402808012801"),
		}
		root3Alpha := uint256.MustFromDecimal("15321434214129571")
		expected := uint256.MustFromDecimal("72763504628587906")
		actual, err := Gyro3CLPMath._calculateInvariant(balances, root3Alpha)
		assert.Nil(t, err)
		assert.Equal(t, expected.Dec(), actual.Dec())
	})

	t.Run("5. should return correct result", func(t *testing.T) {
		balances := []*uint256.Int{
			uint256.MustFromDecimal("4432163212310430512"),
			uint256.MustFromDecimal("134122147042142150214"),
			uint256.MustFromDecimal("54231412421402808012801"),
		}
		root3Alpha := uint256.MustFromDecimal("35321434214129571")
		expected := uint256.MustFromDecimal("599894531547570238671")
		actual, err := Gyro3CLPMath._calculateInvariant(balances, root3Alpha)
		assert.Nil(t, err)
		assert.Equal(t, expected.Dec(), actual.Dec())
	})

	t.Run("6. should return correct result", func(t *testing.T) {
		balances := []*uint256.Int{
			uint256.MustFromDecimal("443216321231021430512"),
			uint256.MustFromDecimal("1341221473042142150214"),
			uint256.MustFromDecimal("542314124214028408012801"),
		}
		root3Alpha := uint256.MustFromDecimal("354214129571")
		expected := uint256.MustFromDecimal("61886509378097866327610430")
		actual, err := Gyro3CLPMath._calculateInvariant(balances, root3Alpha)
		assert.Nil(t, err)
		assert.Equal(t, expected.Dec(), actual.Dec())
	})

	t.Run("7. should return correct result", func(t *testing.T) {
		balances := []*uint256.Int{
			uint256.MustFromDecimal("21739217493251"),
			uint256.MustFromDecimal("2147132108591043214"),
			uint256.MustFromDecimal("532194721947251"),
		}
		root3Alpha := uint256.MustFromDecimal("99999999")
		expected := uint256.MustFromDecimal("13752014286181453537732")
		actual, err := Gyro3CLPMath._calculateInvariant(balances, root3Alpha)
		assert.Nil(t, err)
		assert.Equal(t, expected.Dec(), actual.Dec())
	})

	t.Run("8. should return error", func(t *testing.T) {
		balances := []*uint256.Int{
			uint256.MustFromDecimal("21739217493251"),
			uint256.MustFromDecimal("2147132108591043214"),
			uint256.MustFromDecimal("532194721947251"),
		}
		root3Alpha := uint256.MustFromDecimal("421421499999999131343214")
		_, err := Gyro3CLPMath._calculateInvariant(balances, root3Alpha)
		assert.Equal(t, math.ErrSqrtFailed, err)
	})
}
