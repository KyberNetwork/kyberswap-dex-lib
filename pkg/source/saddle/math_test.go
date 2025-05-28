package saddle

import (
	"math/big"
	"testing"

	"github.com/sirupsen/logrus"

	constant "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestCalculateSwap(t *testing.T) {
	t.Parallel()
	nTokens := 3
	var tokenBalances = []*big.Int{
		utils.NewBig10("263829355937719884193312195"),
		utils.NewBig10("253026496012159712668944640"),
		utils.NewBig10("215587136525270574976035964"),
	}
	var tokenPrecisions = []*big.Int{
		utils.NewBig10("1"),
		utils.NewBig10("1"),
		utils.NewBig10("1"),
	}

	balances := make([]*big.Int, 0)
	precisions := make([]*big.Int, 0)
	for i := 0; i < nTokens; i++ {
		balances = append(balances, tokenBalances[i])
		precisions = append(precisions, tokenPrecisions[i])
	}
	var amountIn = utils.NewBig10("10000000000000000000000")

	initialA := big.NewInt(1500)
	futureA := big.NewInt(1500)
	initialATime := int64(0)
	futureATime := int64(0)
	swapFee := big.NewInt(4000000)
	var amountOut, fee, _ = _calculateSwap(
		balances,
		precisions,
		futureATime,
		futureA,
		initialATime,
		initialA,
		swapFee,
		0,
		2,
		amountIn,
	)
	logrus.Info(amountOut.String())
	logrus.Info(fee.String())
}

func TestCalculateSwap2(t *testing.T) {
	t.Parallel()
	nTokens := 2
	var tokenBalances = []*big.Int{
		utils.NewBig10("300524667948860812161452556"),
		utils.NewBig10("307909381422032691082033859"),
	}
	var tokenPrecisions = []*big.Int{
		utils.NewBig10("1"),
		utils.NewBig10("1"),
	}

	balances := make([]*big.Int, 0)
	precisions := make([]*big.Int, 0)
	for i := 0; i < nTokens; i++ {
		balances = append(balances, tokenBalances[i])
		precisions = append(precisions, tokenPrecisions[i])
	}
	var amountIn = utils.NewBig10("10000000000000000000000")

	initialA := big.NewInt(60000)
	futureA := big.NewInt(60000)
	initialATime := int64(0)
	futureATime := int64(0)
	swapFee := big.NewInt(4000000)
	var amountOut, fee, _ = _calculateSwap(
		balances,
		precisions,
		futureATime,
		futureA,
		initialATime,
		initialA,
		swapFee,
		1,
		0,
		amountIn,
	)
	logrus.Info(amountOut.String())
	logrus.Info(fee.String())
}

func TestCalculateWithdrawOneToken(t *testing.T) {
	t.Parallel()
	var balances = []*big.Int{
		utils.NewBig10("264038322528061367790859241"),
		utils.NewBig10("253311544042014270158626065"),
		utils.NewBig10("216304364840015899371623343"),
	}
	var tokenPrecisionMultipliers = []*big.Int{
		utils.NewBig10("1"),
		utils.NewBig10("1"),
		utils.NewBig10("1"),
	}
	lpSupply := utils.NewBig10("706595268772543216633613610")
	initialA := big.NewInt(1500)
	futureA := big.NewInt(1500)
	initialATime := int64(0)
	futureATime := int64(0)
	swapFee := big.NewInt(4000000)
	nCoins := big.NewInt(3)
	withdrawFee := new(big.Int).Div(new(big.Int).Mul(swapFee, nCoins), new(big.Int).Mul(constant.Four, new(big.Int).Sub(nCoins, constant.One)))
	logrus.Info(withdrawFee.String())
	amount := utils.NewBig10("10000000000000000000000")

	dy, dySwapFee, err := calculateWithdrawOneToken(
		balances,
		tokenPrecisionMultipliers,
		futureATime,
		futureA,
		initialATime,
		initialA,
		swapFee,
		constant.ZeroBI,
		//vmath.NewBig10("5000000"),
		lpSupply,
		0,
		amount,
	)
	// 10381470568495055373796
	// 10381470568495055373796

	if err != nil {
		logrus.Error(err)
	} else {
		logrus.Info(dy.String(), " ", dySwapFee.String())
	}
}

func TestCalculateTokenAmount(t *testing.T) {
	t.Parallel()
	var balances = []*big.Int{
		utils.NewBig10("1762846108183174838018939"),
		utils.NewBig10("3674225304303"),
		utils.NewBig10("3196888988762"),
	}
	var tokenPrecisionMultipliers = []*big.Int{
		utils.NewBig10("1"),
		utils.NewBig10("1000000000000"),
		utils.NewBig10("1000000000000"),
	}
	dy, err := calculateTokenAmount(
		balances,
		tokenPrecisionMultipliers,
		0,
		big.NewInt(80000),
		0,
		big.NewInt(80000),
		utils.NewBig10("5000000"),
		utils.NewBig10("8580021119487881426822908"),
		[]*big.Int{
			utils.NewBig10("0"),
			utils.NewBig10("0"),
			utils.NewBig10("1000000"),
		},
		true,
	)

	if err != nil {
		logrus.Error(err)
	} else {
		logrus.Info(dy.String())
	}
}

func TestGetDyUnderlying(t *testing.T) {
	t.Parallel()
	var balances = []*big.Int{
		utils.NewBig10("70382246141748845587674511"),
		utils.NewBig10("164292114057107"),
		utils.NewBig10("205974869965084"),
	}
	var tokenPrecisionMultipliers = []*big.Int{
		utils.NewBig10("1"),
		utils.NewBig10("1000000000000"),
		utils.NewBig10("1000000000000"),
	}

	var tokenIndexFrom = 0
	var tokenIndexTo = 2
	var dx = utils.NewBig10("1234567890000000000")

	dy, fee, err := GetDyUnderlying(
		balances,
		tokenPrecisionMultipliers,
		1621013782,
		big.NewInt(200000),
		1620408998,
		big.NewInt(100000),
		big.NewInt(3000000),
		big.NewInt(20000000000),
		tokenIndexFrom,
		tokenIndexTo,
		dx,
	)

	if err != nil {
		logrus.Error(err)
	} else {
		logrus.Info(dy.String(), " ", fee.String())
	}
}
