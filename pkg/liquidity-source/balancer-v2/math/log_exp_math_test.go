package math

import (
	"fmt"
	"testing"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestLogExpMath_Pow(t *testing.T) {
	t.Run("it should return correct result", func(t *testing.T) {
		result, err := LogExpMath.Pow(number.NewUint256("999955774281269788"), number.NewUint256("4000000000000000000"))

		fmt.Printf("result: %s\n", result.ToBig().String())

		assert.Nil(t, err)
		assert.Zero(t, result.Cmp(number.NewUint256("999823108860218333")))
	})
}

func TestLogExpMath_Exp(t *testing.T) {
	t.Run("should be return correct result", func(t *testing.T) {
		result, err := LogExpMath.Exp(bignumber.NewBig10("-176906786864581"))

		assert.Nil(t, err)
		assert.Zero(t, result.Cmp(bignumber.NewBig10("999823108860218333")))
	})
}

func TestLogExpMath_ln_36(t *testing.T) {
	t.Run("it should return correct result", func(t *testing.T) {
		result := LogExpMath._ln_36(bignumber.NewBig10("999955774281269788"))

		assert.Zero(t, result.Cmp(bignumber.NewBig10("-44226696716145462061506341909418")))
	})
}
