package spfav2

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
)

func Test_BestRoute(t *testing.T) {
	t.Skip("Skip due to new pregen kpath algo implementation, should impl another testcase")

	maxHops := uint32(3)
	distributionPercent := uint32(5)
	maxPathsInRoute := uint32(20)
	maxPathsToGenerate := uint32(5)
	maxPathsToReturn := uint32(200)
	minPartUSD := float64(500)
	minThresholdAmountInUSD := float64(0)
	maxThresholdAmountInUSD := float64(100000000)

	finder := NewSPFAv2Finder(
		maxHops,
		distributionPercent,
		maxPathsInRoute,
		maxPathsToGenerate,
		maxPathsToReturn,
		minPartUSD,
		minThresholdAmountInUSD,
		maxThresholdAmountInUSD,
		nil,
	)
	input := findroute.Input{
		TokenInAddress:  "0x69b2cd28b205b47c8ba427e111dd486f9c461b57",
		TokenOutAddress: "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
		AmountIn:        big.NewInt(19),
		SaveGas:         false,
		GasInclude:      true,
	}

	data := findroute.FinderData{
		TokenByAddress: map[string]entity.Token{
			"0x69b2cd28b205b47c8ba427e111dd486f9c461b57": {
				Decimals: 0,
			},
		},
		PriceUSDByAddress: map[string]float64{
			"0x69b2cd28b205b47c8ba427e111dd486f9c461b57": 64.64146875432759,
		},
	}

	totalAmountIn := poolpkg.TokenAmount{
		Token:     "0x69b2cd28b205b47c8ba427e111dd486f9c461b57",
		Amount:    big.NewInt(19),
		AmountUsd: 1228.1879063322242,
	}

	assert.NotPanics(t, func() { finder.splitAmountIn(input, data, totalAmountIn) })
}
