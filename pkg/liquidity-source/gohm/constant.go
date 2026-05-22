package gohm

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeGOHM

	defaultReserve = "1000000000000000000"
)

type Action uint8

const (
	ActionStakeToSOHM Action = iota // OHM -> sOHM (stake rebasing=true)
	ActionUnstakeSOHM               // sOHM -> OHM (unstake rebasing=true)
	ActionStakeToGOHM               // OHM -> gOHM (stake rebasing=false)
	ActionUnstakeGOHM               // gOHM -> OHM (unstake rebasing=false)
	ActionWrap                      // sOHM -> gOHM
	ActionUnwrap                    // gOHM -> sOHM
)

var dfGas = Gas{
	Stake:   200_000,
	Unstake: 200_000,
	Wrap:    150_000,
	Unwrap:  150_000,
}

var (
	ErrInvalidTokenIn        = errors.New("invalid tokenIn")
	ErrInvalidTokenOut       = errors.New("invalid tokenOut")
	ErrWarmupActive          = errors.New("warmupPeriod > 0: staking is not atomic")
	ErrZeroAmount            = errors.New("zero amount in")
	ErrIndexZero             = errors.New("gOHM index is zero")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity in staking contract")
)
