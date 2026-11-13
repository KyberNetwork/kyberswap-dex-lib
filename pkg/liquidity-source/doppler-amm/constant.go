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

// Deployment pins one DopplerAMM deployment set. Base and Base Sepolia run the
// same addresses, so only the deployment block differs.
type Deployment struct {
	AMM          string
	SpotModule   string
	LaunchModule string
	FeesLocker   string // the only extension a supported Launch Market carries
	Quoter       string
	StartBlock   uint64
}

var deployments = map[int]Deployment{
	8453: {
		AMM:          "0x533b7ee11f4f9be09191ebf6667abe73c01fe2c2",
		SpotModule:   "0x3d17adc45d3c40735e7fdd271710f90d84c1b838",
		LaunchModule: "0xf2e28140520de30de998ea25f46923bb7bebd909",
		FeesLocker:   "0x05cfa684b6f28c3dfddcd6f9b361c8f90cd09ead",
		Quoter:       "0x384e4c09071f7e511be34e38b4fcba23be51ea55",
		StartBlock:   52856797,
	},
	84532: {
		AMM:          "0x533b7ee11f4f9be09191ebf6667abe73c01fe2c2",
		SpotModule:   "0x3d17adc45d3c40735e7fdd271710f90d84c1b838",
		LaunchModule: "0xf2e28140520de30de998ea25f46923bb7bebd909",
		FeesLocker:   "0x05cfa684b6f28c3dfddcd6f9b361c8f90cd09ead",
		Quoter:       "0x384e4c09071f7e511be34e38b4fcba23be51ea55",
		StartBlock:   48399444,
	},
}

var (
	ErrUnsupportedChain      = errors.New("doppler-amm: unsupported chain")
	ErrUnsupportedMarket     = errors.New("doppler-amm: unsupported market composition")
	ErrInsufficientLiquidity = errors.New("doppler-amm: insufficient liquidity")
	ErrInvalidToken          = errors.New("doppler-amm: invalid token")
	ErrInvalidAmountIn       = errors.New("doppler-amm: invalid amount in")
	ErrInsufficientOutput    = errors.New("doppler-amm: insufficient output")
	ErrFeeDerivation         = errors.New("doppler-amm: cannot resolve market fee from quote probe")
	ErrCallReverted          = errors.New("doppler-amm: rpc call reverted")
	ErrOverflow              = errors.New("doppler-amm: overflow")
	ErrLaunchState           = errors.New("doppler-amm: launch market state inconsistent with its levels")
)

var (
	bpsDenomU256 = uint256.NewInt(bpsDenominator)
)
