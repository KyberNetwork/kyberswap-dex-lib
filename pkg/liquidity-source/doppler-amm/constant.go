package doppleramm

import (
	"errors"

	"github.com/holiman/uint256"
)

const (
	DexType = "doppler-amm"

	// defaultGas covers an exact-input swap through DopplerAMM plus the
	// executor's approve. Measured 113.6k for the swap itself on a Base
	// Sepolia fork (cold storage, 1e18 in); the rest is headroom for the
	// approve and balance check the generic executor adds around it.
	defaultGas = 140_000

	bpsDenominator = 10_000
)

var (
	ErrInvalidConfig         = errors.New("doppler-amm: invalid config address")
	ErrUnsupportedMarket     = errors.New("doppler-amm: unsupported market composition")
	ErrInsufficientLiquidity = errors.New("doppler-amm: insufficient liquidity")
	ErrInvalidToken          = errors.New("doppler-amm: invalid token")
	ErrInvalidAmountIn       = errors.New("doppler-amm: invalid amount in")
	ErrInsufficientOutput    = errors.New("doppler-amm: insufficient output")
	ErrCallReverted          = errors.New("doppler-amm: rpc call reverted")
	ErrOverflow              = errors.New("doppler-amm: overflow")
	ErrLaunchState           = errors.New("doppler-amm: launch market state inconsistent with its levels")
)

var (
	bpsDenomU256 = uint256.NewInt(bpsDenominator)
)
