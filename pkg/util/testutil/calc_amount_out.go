package testutil

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// TestCalcAmountOut tests CalcAmountOut with given input and output amounts. Empty output means error expected
func TestCalcAmountOut[TB interface {
	testing.TB
	Run(string, func(TB)) bool
}](tb TB, poolSim pool.IPoolSimulator, expected map[int]map[int]map[string]string) {
	tokens := poolSim.GetTokens()
	for idxIn, expectedByTokenIn := range expected {
		for idxOut, expectedByTokenOut := range expectedByTokenIn {
			for amtIn, expectedAmount := range expectedByTokenOut {
				tb.Run(fmt.Sprintf("%s token%d -> ? token%d", amtIn, idxIn, idxOut), func(tb TB) {
					amtOut, err := pool.CalcAmountOut(
						poolSim,
						pool.TokenAmount{Token: tokens[idxIn], Amount: bignumber.NewBig10(amtIn)},
						tokens[idxOut],
						nil,
					)
					if expectedAmount == "" {
						assert.Error(tb, err)
					} else if assert.NoError(tb, err) {
						assert.Equal(tb, expectedAmount, amtOut.TokenAmountOut.Amount.String())
					}
				})
			}
		}
	}
}
