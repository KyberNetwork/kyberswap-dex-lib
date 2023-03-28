package bruteforce

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	poolpkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
)

func TestBruteforceFinder_GenerateSplits(t *testing.T) {
	// minPartUSD = 500
	var tests = []struct {
		amountIn          int
		splittedAmountIns [][]int
	}{
		{400, [][]int{{400}}},
		{700, [][]int{{700}}},
		{1400, [][]int{{560, 840}, {840, 560}, {630, 770}, {770, 630}, {700, 700}, {1400}}},
		{1005, [][]int{{502, 503}, {1005}}},
	}
	finder := NewDefaultBruteforceFinder(nil, nil)
	input := findroute.Input{
		TokenInAddress: "tokenIn",
		SaveGas:        false,
	}
	data := findroute.FinderData{
		PriceUSDByAddress: map[string]float64{
			"tokenIn": 1,
		},
		TokenByAddress: map[string]entity.Token{
			"tokenIn": {Decimals: 0},
		},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("test split amount in %v", test.amountIn), func(t *testing.T) {
			tokenAmountIn := poolpkg.TokenAmount{
				Token:     "tokenIn",
				Amount:    big.NewInt(int64(test.amountIn)),
				AmountUsd: float64(test.amountIn),
			}
			splits, err := finder.generateSplits(input, data, tokenAmountIn)
			assert.Nil(t, err)
			assert.Equal(t, len(test.splittedAmountIns), len(splits))
			for i := 0; i < len(splits); i++ {
				assert.Equal(t, len(test.splittedAmountIns[i]), len(splits[i]))
				for j := range splits[i] {
					assert.Equal(t, splits[i][j].Amount.Cmp(big.NewInt(int64(test.splittedAmountIns[i][j]))), 0)
				}
			}
		})
	}
}
