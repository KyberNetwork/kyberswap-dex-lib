package netstaking

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeNetStaking

	defaultReserve = "1000000000000000000"

	// Default getter names - used when Config leaves BaseTokenMethod/StakedTokenMethod
	// unset, keeping existing net-staking configs unchanged.
	defaultBaseTokenMethod   = "net"
	defaultStakedTokenMethod = "sNet"
)

type Action uint8

const (
	ActionStake        Action = iota // NET -> sNET (Staking.stake, 1:1)
	ActionUnstake                    // sNET -> NET (Staking.unstake, 1:1)
	ActionWrap                       // sNET -> wsNET (WrappedStakedNET.wrap, ratio via index())
	ActionUnwrap                     // wsNET -> sNET (WrappedStakedNET.unwrap, ratio via index())
	ActionStakeAndWrap               // NET -> wsNET (composite: stake then wrap)
)

var dfGas = Gas{
	Stake:        42258,
	Unstake:      40162,
	Wrap:         37900,
	Unwrap:       37739,
	StakeAndWrap: 42258 + 37900,
}

var (
	ErrInvalidTokenIn        = errors.New("invalid tokenIn")
	ErrInvalidTokenOut       = errors.New("invalid tokenOut")
	ErrZeroAmount            = errors.New("zero amount in")
	ErrIndexZero             = errors.New("sNET index is zero")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity in staking/wrap contract")
)
