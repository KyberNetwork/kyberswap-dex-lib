package eethorweeth

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type PoolExtra struct {
	StETH struct {
		TotalPooledEther *big.Int
		TotalShares      *big.Int
	}

	StETHTokenInfo struct {
		DiscountInBasisPoints      uint16
		TotalDepositedThisPeriod   *big.Int
		TotalDeposited             *big.Int
		TimeBoundCapClockStartTime uint32
		TimeBoundCapInEther        uint32
		TotalCapInEther            uint32
	}

	Vampire struct {
		QuoteStEthWithCurve         bool
		TimeBoundCapRefreshInterval uint32
	}

	EtherFiPool struct {
		TotalPooledEther *big.Int
	}

	EETH struct {
		TotalShares *big.Int
	}

	CurveStETHToETH CurvePoolInfo
}

type CurvePoolInfo struct {
	Reserves    []string
	Extra       string
	StaticExtra string
}

type CurvePlainExtra struct {
	InitialA     *big.Int
	FutureA      *big.Int
	InitialATime *big.Int
	FutureATime  *big.Int
	SwapFee      *big.Int
	AdminFee     *big.Int
}

type VampireTokenInfo struct {
	StrategyShare                  *big.Int
	EthAmountPendingForWithdrawals *big.Int
	Strategy                       common.Address
	IsWhitelisted                  bool
	DiscountInBasisPoints          uint16
	TimeBoundCapClockStartTime     uint32
	TimeBoundCapInEther            uint32
	TotalCapInEther                uint32
	TotalDepositedThisPeriod       *big.Int
	TotalDeposited                 *big.Int
	IsL2Eth                        bool
}
