package gblin

import (
	"errors"

	"github.com/holiman/uint256"
)

const (
	DexType = "gblin"

	vaultMethodTotalSupply   = "totalSupply"
	vaultMethodTotalEthValue = "totalEthValue"
	vaultMethodIsNavReliable = "isNavReliable"

	lensMethodConfigFees               = "configFees"
	lensMethodManagementFeeBps         = "managementFeeBps"
	lensMethodLastManagementFeeAccrual = "lastManagementFeeAccrual"
	lensMethodSequencerFeed            = "sequencerFeed"

	aggregatorMethodLatestRoundData = "latestRoundData"

	multicallMethodGetEthBalance            = "getEthBalance"
	multicallMethodGetCurrentBlockTimestamp = "getCurrentBlockTimestamp"

	reserveZero = "0"

	secondsPerYear = 365 * 24 * 60 * 60

	// sequencerGracePeriod mirrors GBLIN.SEQUENCER_GRACE_PERIOD: mints revert for this long after the
	// sequencer comes back up.
	sequencerGracePeriod = 60 * 60

	// defaultMintGas is the gas used by buyGBLINWithWeth on Base: 506,004 to 543,656 over eight mints read
	// from the explorer. The cost does not depend on the amount; it varies with the basket's oracle reads.
	defaultMintGas int64 = 560_000
)

var (
	bps           = uint256.NewInt(10_000)
	year          = uint256.NewInt(secondsPerYear)
	virtualShares = uint256.NewInt(1_000_000)
	virtualAssets = uint256.NewInt(1)

	ErrInvalidToken       = errors.New("invalid token")
	ErrUnsupportedSwap    = errors.New("only WETH -> GBLIN (exact in) is supported")
	ErrInvalidAmountIn    = errors.New("invalid amount in")
	ErrDepositTooSmall    = errors.New("deposit below the vault's minimum")
	ErrNavUnreliable      = errors.New("vault NAV is not reliable: mints revert")
	ErrSequencerDown      = errors.New("sequencer down or within grace period: mints revert")
	ErrStateUnavailable   = errors.New("vault state unavailable")
	ErrZeroAmountOut      = errors.New("zero amount out")
	ErrFeeExceedsAmountIn = errors.New("fees exceed amount in")
)
