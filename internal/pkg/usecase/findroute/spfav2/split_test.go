package spfav2

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/blockchain-toolkit/float"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/pool-service/pkg/util/bignumber"
	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func TestSplit(t *testing.T) {

	token := "X"
	tokenDecimal := uint8(6)
	tokenPriceUSD := float64(1000) // 1 X -> 1k usd

	gasToken := "gas"
	nativePriceInUSD := 3649.21                                        // 1 eth -> 3649.21 usd
	tokenPriceNative := big.NewFloat(tokenPriceUSD / nativePriceInUSD) // 1 X -> 1k usd -> (1k/3649.21) eth
	// 1 wei of X -> (1k*10^18/3649.21/10^6) wei of eth
	tokenPriceNativeRaw := new(big.Float).Mul(
		tokenPriceNative, float.TenPow(18-tokenDecimal),
	)

	minPartUSD := float64(500)
	distPercent := uint32(5)

	input := findroute.Input{TokenInAddress: token, GasTokenPriceUSD: nativePriceInUSD}
	data := findroute.FinderData{
		PriceUSDByAddress: map[string]float64{token: tokenPriceUSD},
		TokenByAddress: map[string]*entity.Token{
			token: {Decimals: tokenDecimal},
		},
	}
	dataWithNative := findroute.FinderData{
		PriceUSDByAddress: map[string]float64{gasToken: nativePriceInUSD},
		PriceNativeByAddress: map[string]*routerEntity.OnchainPrice{
			token: {
				NativePrice:    routerEntity.Price{Sell: tokenPriceNative},
				NativePriceRaw: routerEntity.Price{Sell: tokenPriceNativeRaw},
			},
		},
		TokenByAddress: map[string]*entity.Token{
			token: {Decimals: tokenDecimal},
		},
	}

	finder := NewSPFAv2Finder(
		3,
		nil,
		distPercent,
		20,
		5,
		200,
		minPartUSD,
		0,
		float64(100000000),
		map[string]bool{},
	)

	testcases := []struct {
		name           string
		amountIn       string
		expectedSplits []string
	}{
		{"1 dollar -> 1x1", "1000", []string{"1000"}},
		{"1k dollar -> 2x500", "1000000", []string{"500000", "500000"}},
		{"1.2k dollar -> 660,540", "1200000", []string{"660000", "540000"}},
		{"1.5k dollar -> 975,525", "1500000", []string{"975000", "525000"}},
		{"10k dollar -> 20x500", "10000000", []string{"500000", "500000", "500000", "500000", "500000", "500000", "500000", "500000", "500000", "500000", "500000", "500000", "500000", "500000", "500000", "500000", "500000", "500000", "500000", "500000"}},
		{"15k dollar -> 20x750", "15000000", []string{"750000", "750000", "750000", "750000", "750000", "750000", "750000", "750000", "750000", "750000", "750000", "750000", "750000", "750000", "750000", "750000", "750000", "750000", "750000", "750000"}},
	}

	for _, tc := range testcases {
		ain := bignumber.NewBig10(tc.amountIn)
		t.Run(tc.name+" - use USD", func(t *testing.T) {
			amounts := finder.splitAmountIn(
				input, data,
				valueobject.TokenAmount{
					Token:  token,
					Amount: ain,
					AmountUsd: utils.CalcTokenAmountUsd(
						ain,
						data.TokenByAddress[token].Decimals,
						data.PriceUSDByAddress[token],
					),
				},
			)

			amountStrs := lo.Map(amounts, func(a valueobject.TokenAmount, _ int) string { return a.Amount.String() })
			assert.Equal(t, tc.expectedSplits, amountStrs)
		})

		t.Run(tc.name+" - use Native", func(t *testing.T) {
			amountInNative, _ := new(big.Float).Mul(tokenPriceNativeRaw, new(big.Float).SetInt(ain)).Int(&big.Int{})
			amounts := finder.splitAmountIn(
				input, dataWithNative,
				valueobject.TokenAmount{
					Token:          token,
					Amount:         ain,
					AmountAfterGas: amountInNative,
				},
			)

			amountStrs := lo.Map(amounts, func(a valueobject.TokenAmount, _ int) string { return a.Amount.String() })
			assert.Equal(t, tc.expectedSplits, amountStrs)
		})
	}

}
