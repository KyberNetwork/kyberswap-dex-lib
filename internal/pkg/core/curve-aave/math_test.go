package curveAave

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/utils"

	"github.com/sirupsen/logrus"
)

//func TestCalculateSwap(t *testing.T) {
//	var tokens = []store.PoolToken {
//		{
//			Token:               "0x1af3f329e8be154074d8769d1ffa4ee058b1dbc3",
//			Balance:             utils.NewBig10("5509778105938260159312596"),
//			Weight:              0,
//			Multiplier: utils.One,
//		},
//		{
//			Token:               "0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d",
//			Balance:             utils.NewBig10("6798101710196342168913370"),
//			Weight:              0,
//			Multiplier: utils.One,
//		},
//		{
//			Token:               "0x55d398326f99059ff775485246999027b3197955",
//			Balance:             utils.NewBig10("6851312148755656450649249"),
//			Weight:              0,
//			Multiplier: utils.One,
//		},
//		{
//			Token:               "0xe9e7cea3dedca5984780bafc599bd69add087d56",
//			Balance:             utils.NewBig10("7451611297377057689912146"),
//			Weight:              0,
//			Multiplier: utils.One,
//		},
//	}
//	pool := store.Pool{
//		Address:    "0xabc",
//		ReserveUsd: 0,
//		SwapFee:    big.NewInt(4000000),
//		Exchange:   "firebird-oneswap",
//		Tokens:     tokens,
//		Extra:      store.ExtraProperties{
//			InitialA:           big.NewInt(20000),
//			FutureA:            big.NewInt(20000),
//			InitialATime:       0,
//			FutureATime:        0,
//			AdminFee:           big.NewInt(5000000000),
//			DefaultWithdrawFee: big.NewInt(0),
//		},
//		Checked:    false,
//	}
//	balances := make([]*big.Int, 0)
//	precisions := make([]*big.Int, 0)
//	for _, token := range tokens {
//		balances = append(balances, token.Balance)
//		precisions = append(precisions, token.Multiplier)
//	}
//	var amountIn = utils.NewBig10("10000000000000000000000")
//	var amountOut, _ = CalculateSwap(
//		balances,
//		precisions,
//		pool.Extra.FutureATime,
//		pool.Extra.FutureA,
//		pool.Extra.InitialATime,
//		pool.Extra.InitialA,
//		pool.SwapFee,
//		0,
//		3,
//		amountIn,
//	)
//	logrus.Info(amountOut.String())
//}

func TestCalculateWithdrawOneToken(t *testing.T) {
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
	dy, dySwapFee, err := calculateWithdrawOneToken(
		balances,
		tokenPrecisionMultipliers,
		0,
		big.NewInt(80000),
		0,
		big.NewInt(80000),
		utils.NewBig10("2000000"),
		utils.NewBig10("5000000"),
		utils.NewBig10("8580021119487881426822908"),
		0,
		utils.NewBig10("10000000000000000000"),
	)

	if err != nil {
		logrus.Error(err)
	} else {
		logrus.Info(dy.String(), " ", dySwapFee.String())
	}
}

func TestCalculateTokenAmount(t *testing.T) {
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
