package prop

import (
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/kipseli"
)

const (
	DexType    = "kipseli-prop"
	defaultGas = 125_000

	// maxAge bounds how long a probed ladder may be quoted against before a
	// fresher one is required — mirrors titan-prop's freshnessTTL.
	maxAge = 30 * time.Second
)

// defaultDest is the KyberSwap executor, the `dest` kipseli keys its
// per-taker quoters by (QuoteRouter.signerToDest on the prop venues).
var defaultDest = common.HexToAddress("0x8f10b468b06c6fd214b65f87778827f7d113f996")

var (
	ErrInvalidToken          = kipseli.ErrInvalidToken
	ErrInsufficientLiquidity = kipseli.ErrInsufficientLiquidity

	ErrUnexpectedLensRevert = errors.New("kipseli-prop: lens reverted without a snapshot")
	ErrNoQuoter             = errors.New("kipseli-prop: no quoter resolved for dest")
)
