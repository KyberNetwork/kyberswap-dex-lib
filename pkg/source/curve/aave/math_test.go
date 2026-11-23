package aave

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/sirupsen/logrus"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

func TestCalculateWithdrawOneToken(t *testing.T) {
	t.Parallel()
	var balances = []*uint256.Int{
		big256.New("1762846108183174838018939"),
		big256.New("3674225304303"),
		big256.New("3196888988762"),
	}
	var tokenPrecisionMultipliers = []*uint256.Int{
		big256.New("1"),
		big256.New("1000000000000"),
		big256.New("1000000000000"),
	}
	dy, dySwapFee, err := calculateWithdrawOneTokenDy(
		balances,
		tokenPrecisionMultipliers,
		0,
		uint256.NewInt(80000),
		0,
		uint256.NewInt(80000),
		big256.New("2000000"),
		big256.New("8580021119487881426822908"),
		0,
		big256.New("10000000000000000000"),
	)

	if err != nil {
		logrus.Error(err)
	} else {
		logrus.Info(dy.String(), " ", dySwapFee.String())
	}
}

func TestCalculateTokenAmount(t *testing.T) {
	t.Parallel()
	var balances = []*uint256.Int{
		big256.New("1762846108183174838018939"),
		big256.New("3674225304303"),
		big256.New("3196888988762"),
	}
	var tokenPrecisionMultipliers = []*uint256.Int{
		big256.New("1"),
		big256.New("1000000000000"),
		big256.New("1000000000000"),
	}
	dy, err := calculateTokenAmount(
		balances,
		tokenPrecisionMultipliers,
		0,
		uint256.NewInt(80000),
		0,
		uint256.NewInt(80000),
		big256.New("5000000"),
		big256.New("8580021119487881426822908"),
		[]*uint256.Int{
			big256.New("0"),
			big256.New("0"),
			big256.New("1000000"),
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
	var balances = []*uint256.Int{
		big256.New("70382246141748845587674511"),
		big256.New("164292114057107"),
		big256.New("205974869965084"),
	}
	var tokenPrecisionMultipliers = []*uint256.Int{
		big256.New("1"),
		big256.New("1000000000000"),
		big256.New("1000000000000"),
	}

	var tokenIndexFrom = 0
	var tokenIndexTo = 2
	var dx = big256.New("1234567890000000000")

	dy, fee, err := GetDyUnderlying(
		balances,
		tokenPrecisionMultipliers,
		1621013782,
		uint256.NewInt(200000),
		1620408998,
		uint256.NewInt(100000),
		big256.New("3000000"),
		big256.New("20000000000"),
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
