package hyperamm

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeHyperAMM

	defaultGas = int64(536883)
)

var (
	Router     = "0x18ebC3F95CD6B5db55A15864079019C5d2b83DBC"
	RouterAddr = common.HexToAddress(Router)

	ErrPoolPaused          = errors.WithMessage(pool.ErrUnsupported, "hyperamm: pool is paused")
	ErrInvalidToken        = errors.New("hyperamm: invalid token")
	ErrZeroAmountIn        = errors.New("hyperamm: zero amount in")
	ErrZeroAmountOut       = errors.New("hyperamm: zero amount out")
	ErrInsufficientReserve = errors.New("hyperamm: insufficient reserve")
	ErrZeroFairPrice       = errors.New("hyperamm: zero fair price")
)
