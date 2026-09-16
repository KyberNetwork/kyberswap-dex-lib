package lunyafun

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type Gas struct {
	BaseGas int64
}

// Extra is a launch's curve state, all read at one block. Everything below Sold is fixed at creation,
// but it is read with the rest: discovery is a pure log decode, so this is the first point it is known.
type Extra struct {
	Phase   uint8        `json:"phase"`
	Reserve *uint256.Int `json:"reserve"`
	Sold    *uint256.Int `json:"sold"`

	VirtualQuote *uint256.Int `json:"virtualQuote"`
	VirtualToken *uint256.Int `json:"virtualToken"`
	CurveSupply  *uint256.Int `json:"curveSupply"`
	CurveFeeBps  uint16       `json:"curveFeeBps"`

	// The anti-snipe surcharge: a tax on top of the curve fee that decays to nothing over the window
	// after the launch opened, and which exempt recipients never pay.
	SnipeTaxBps uint16 `json:"snipeTaxBps,omitempty"`
	SnipeWindow uint32 `json:"snipeWindow,omitempty"`
	SnipeDecay  uint8  `json:"snipeDecay,omitempty"`
	OpenedAt    uint64 `json:"openedAt,omitempty"`
}

type StaticExtra struct {
	LaunchType uint8 `json:"launchType"`
}

type SwapInfo struct {
	NextReserve *uint256.Int `json:"-"`
	NextSold    *uint256.Int `json:"-"`
}

// PoolMeta names the launch as the contract to approve: a buy pulls the quote token and a sell pulls
// the launched token, both with transferFrom. IsBuy tells the encoder which of the launch's two calls
// to build, since token0/token1 order is not itself buy/sell direction.
type PoolMeta struct {
	ApprovalAddress string `json:"approvalAddress"`
	IsBuy           bool   `json:"isBuy"`
	BlockNumber     uint64 `json:"blockNumber"`
}

// launchConfigResp mirrors ILunyaLaunch.Config, which config() returns whole.
type launchConfigResp struct {
	Factory             common.Address
	QuoteToken          common.Address
	Creator             common.Address
	CreatorFeeRecipient common.Address
	PositionManager     common.Address
	LiquidityHelper     common.Address
	Locker              common.Address
	NativeDivisor       *big.Int
	WrapsNative         bool
	TotalSupply         *big.Int
	VirtualQuote        *big.Int
	VirtualToken        *big.Int
	CurveSupply         *big.Int
	LpSupply            *big.Int
	SnipeTaxBps         uint16
	SnipeWindow         uint32
	SnipeDecay          uint8
	GraduationReward    *big.Int
	CurveFeeBps         uint16
	CurveFeeProtocolBps uint16
	GraduationFeeBps    uint16
	PoolFeeProtocolBps  uint16
	GraduationPoolType  uint8
}
