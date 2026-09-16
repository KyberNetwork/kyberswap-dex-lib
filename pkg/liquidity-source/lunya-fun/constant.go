package lunyafun

import (
	"errors"
)

const DexType = "lunya-fun"

// ILunyaLaunch.Phase. A launch trades on its curve only while Trading; past that its liquidity has
// moved to a Lunya DEX pool, which the `lunya` source prices.
const (
	phaseNone           = 0
	phaseTrading        = 1
	phaseReadyToGradate = 2
	phaseGraduated      = 3
)

// LaunchTypes.CONSTANT_PRODUCT, the curve this source prices. LaunchType is a uint8 the factory hands
// out per implementation, and zero is no type at all.
const launchTypeCP = 1

// bps is the launchpad's fee denominator.
const bps = 10_000

const (
	launchMethodPhase    = "phase"
	launchMethodToken    = "token"
	launchMethodReserve  = "reserve"
	launchMethodSold     = "sold"
	launchMethodOpenedAt = "openedAt"
	launchMethodConfig   = "config"
)

// Measured through ks-dex-adapter-lib's LunyaFunAdapter on an Arc testnet fork, cold storage:
// 106k-134k per trade, the high end being a buy, which pulls the quote token and mints out of the curve.
var defaultGas = Gas{BaseGas: 135_000}

var (
	ErrInvalidToken        = errors.New("invalid token")
	ErrInvalidAmount       = errors.New("invalid amount")
	ErrZeroAmount          = errors.New("zero amount")
	ErrNotTrading          = errors.New("launch is not on its curve")
	ErrMoreThanSold        = errors.New("more tokens than the curve has sold")
	ErrInsufficientReserve = errors.New("insufficient reserve")
	ErrArithmetic          = errors.New("curve arithmetic error")
	ErrUnsupportedLaunch   = errors.New("unsupported launch type")
	ErrMalformedLog        = errors.New("malformed event log")
	ErrFailedCall          = errors.New("launch state call failed")
)
