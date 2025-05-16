package aave

import (
	"math/big"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestCalculateWithdrawOneToken(t *testing.T) {
	t.Parallel()
	var balances = []*big.Int{
		bignumber.NewBig10("1762846108183174838018939"),
		bignumber.NewBig10("3674225304303"),
		bignumber.NewBig10("3196888988762"),
	}
	var tokenPrecisionMultipliers = []*big.Int{
		bignumber.NewBig10("1"),
		bignumber.NewBig10("1000000000000"),
		bignumber.NewBig10("1000000000000"),
	}
	dy, dySwapFee, err := calculateWithdrawOneToken(
		balances,
		tokenPrecisionMultipliers,
		0,
		big.NewInt(80000),
		0,
		big.NewInt(80000),
		bignumber.NewBig10("2000000"),
		bignumber.NewBig10("5000000"),
		bignumber.NewBig10("8580021119487881426822908"),
		0,
		bignumber.NewBig10("10000000000000000000"),
	)

	if err != nil {
		logrus.Error(err)
	} else {
		logrus.Info(dy.String(), " ", dySwapFee.String())
	}
}

func TestCalculateTokenAmount(t *testing.T) {
	t.Parallel()
	var balances = []*big.Int{
		bignumber.NewBig10("1762846108183174838018939"),
		bignumber.NewBig10("3674225304303"),
		bignumber.NewBig10("3196888988762"),
	}
	var tokenPrecisionMultipliers = []*big.Int{
		bignumber.NewBig10("1"),
		bignumber.NewBig10("1000000000000"),
		bignumber.NewBig10("1000000000000"),
	}
	dy, err := calculateTokenAmount(
		balances,
		tokenPrecisionMultipliers,
		0,
		big.NewInt(80000),
		0,
		big.NewInt(80000),
		bignumber.NewBig10("5000000"),
		bignumber.NewBig10("8580021119487881426822908"),
		[]*big.Int{
			bignumber.NewBig10("0"),
			bignumber.NewBig10("0"),
			bignumber.NewBig10("1000000"),
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
		bignumber.NewBig10("70382246141748845587674511"),
		bignumber.NewBig10("164292114057107"),
		bignumber.NewBig10("205974869965084"),
	}
	var tokenPrecisionMultipliers = []*big.Int{
		bignumber.NewBig10("1"),
		bignumber.NewBig10("1000000000000"),
		bignumber.NewBig10("1000000000000"),
	}

	var tokenIndexFrom = 0
	var tokenIndexTo = 2
	var dx = bignumber.NewBig10("1234567890000000000")

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
