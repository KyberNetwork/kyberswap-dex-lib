package smoothy

import (
	"errors"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeSmoothy

	defaultGas = 147745
)

var (
	logUpperBound = uint256.NewInt(1100000000000000000)
	ln2Multiplier = uint256.NewInt(693147180559945344)
)

var (
	ErrInvalidToken                       = errors.New("invalid token")
	ErrZeroSwap                           = errors.New("zero swap")
	ErrMintNewPercentageExceedsHardWeight = errors.New("new percentage exceeds hard weight")
	ErrNegativePenalty                    = errors.New("penalty should be positive")
	ErrOutAmountGreaterThanLPAmount       = errors.New("out amount greater than lp amount")
	ErrInsufficientBalance                = errors.New("insufficient balance")
	ErrCannotFindProperResolutionOfFX     = errors.New("cannot find proper resolution of fx")
	ErrRedeemHardLimitWeightBroken        = errors.New("hard-limit weight is broken")
	ErrLogXInvalidInput                   = errors.New("log(x): x must be greater than or equal to 1")
	ErrLogXTooLarge                       = errors.New("log(x): x is too large")
	ErrLogApproxXMustGteOne               = errors.New("logApprox: x must >= 1")
	ErrLg2XMustBePositive                 = errors.New("lg2: x must be positive")
)
