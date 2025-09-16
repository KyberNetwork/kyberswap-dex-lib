package testutil

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ctx = context.Background()
)

// TestCalcAmountOut tests CalcAmountOut with given input and output amounts. Empty output means error expected
func TestCalcAmountOut[TB interface {
	testing.TB
	Run(string, func(TB)) bool
}](tb TB, poolSim pool.IPoolSimulator, expected map[int]map[int]map[string]string) {
	tb.Helper()
	tokens := poolSim.GetTokens()
	for idxIn, expected := range expected {
		for idxOut, expected := range expected {
			for amtIn, expected := range expected {
				tb.Run(fmt.Sprintf("%s token%d -> ? token%d", amtIn, idxIn, idxOut), func(tb TB) {
					amtOut, err := pool.CalcAmountOut(
						ctx,
						poolSim,
						pool.TokenAmount{Token: tokens[idxIn], Amount: bignumber.NewBig10(amtIn)},
						tokens[idxOut],
						nil,
					)
					if expected == "" {
						require.Error(tb, err)
					} else if expectedAmtOut, ok := new(big.Int).SetString(expected, 0); ok {
						require.NoError(tb, err)
						assert.Equal(tb, expectedAmtOut, amtOut.TokenAmountOut.Amount)
					} else {
						require.EqualError(tb, err, expected)
					}
				})
			}
		}
	}
}
