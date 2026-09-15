package curve

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// StaticExtra is fixed at discovery.
type StaticExtra struct {
	// Hook is the ArcadeHook contract the executor calls: buy(token, amountIn,
	// minTokensOut) and sell(token, tokensIn, minUsdcOut). It pulls the input with
	// transferFrom from the caller and pays the caller.
	Hook string `json:"hook"`
}

// Extra is the per-launch state, refreshed on every tracker pass.
type Extra struct {
	Tracked bool `json:"tr,omitempty"`
	// TokensSold / RealUsdcReserve are CurveState.tokensSold / realUsdcReserve, the
	// only two inputs of ArcadeV4Curve.simulateBuy / simulateSell.
	TokensSold      *uint256.Int `json:"ts"`
	RealUsdcReserve *uint256.Int `json:"ru"`
	Mode            uint8        `json:"m,omitempty"`
	Status          uint8        `json:"s,omitempty"`
	Paused          bool         `json:"p,omitempty"`
	// Anti-sniper curve tax (ArcadeHook.snipeConfigs): a token haircut on buys that
	// decays linearly from SnipeStartBps to 0 over SnipeDecaySeconds after launch.
	SnipeStartBps     uint16 `json:"sb,omitempty"`
	SnipeDecaySeconds uint32 `json:"sd,omitempty"`
	SnipeLaunchedAt   uint64 `json:"sl,omitempty"`
}

// SwapInfo carries what the executor needs and what UpdateBalance applies.
type SwapInfo struct {
	IsBuy bool   `json:"isBuy"`
	Hook  string `json:"hook"`
	Token string `json:"token"`
	// AmountIn is what the hook actually pulls: on a buy that fills the curve it is
	// less than the amount offered, and the rest never leaves the caller.
	AmountIn *big.Int `json:"amountIn"`

	NewTokensSold      *uint256.Int `json:"-"`
	NewRealUsdcReserve *uint256.Int `json:"-"`
	Graduates          bool         `json:"-"`
}

type MetaInfo struct {
	BlockNumber uint64 `json:"blockNumber"`
}

// Decode targets, field order matching the ABI outputs.
type curveStateResult struct {
	VirtualUsdcReserve *big.Int
	RealUsdcReserve    *big.Int
	TokensSold         *big.Int
	Mode               uint8
	Status             uint8
	Creator            common.Address
	Creator2           common.Address
	Creator2Bps        uint16
}

type snipeConfigResult struct {
	StartBps     uint16
	DecaySeconds uint32
	LaunchedAt   uint64
}
