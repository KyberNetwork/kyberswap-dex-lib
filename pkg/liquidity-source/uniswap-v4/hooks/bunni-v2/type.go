package bunniv2

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/oracle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type HookExtra struct {
	HookletExtra        string                `json:"he"`
	HookletAddress      common.Address        `json:"ha"`
	LDFAddress          common.Address        `json:"la"`
	HookFee             *uint256.Int          `json:"hf"`
	PoolManagerReserves [2]*uint256.Int       `json:"pmr"`
	LdfState            [32]byte              `json:"ls"`
	Vaults              [2]Vault              `json:"v"`
	AmAmm               AmAmm                 `json:"aa"`
	ObservationState    ObservationState      `json:"os"`
	CuratorFees         CuratorFees           `json:"cf"`
	Observations        []*oracle.Observation `json:"o"`
	HookParams          HookParams            `json:"hp"`
	Slot0               Slot0                 `json:"s0"`
	BunniState          PoolState             `json:"bs"`
	VaultSharePrices    VaultSharePrices      `json:"vsp"`
	BlockTimestamp      uint32                `json:"bt"`
}

type LdfState struct {
	Initialized bool `json:"i"`
	LastMinTick int  `json:"lmt"`
}

type Vault struct {
	Address    common.Address `json:"a"`
	Decimals   uint8          `json:"d"`
	RedeemRate *uint256.Int   `json:"rr"`
	MaxDeposit *uint256.Int   `json:"md"`
}

type ObservationState struct {
	Index                   uint32              `json:"i"`
	Cardinality             uint32              `json:"c"`
	CardinalityNext         uint32              `json:"cn"`
	IntermediateObservation *oracle.Observation `json:"io"`
}

type VaultSharePrices struct {
	Initialized  bool         `json:"i"`
	SharedPrice0 *uint256.Int `json:"sp0"`
	SharedPrice1 *uint256.Int `json:"sp1"`
}

type CuratorFees struct {
	FeeRate *uint256.Int `json:"fr"`
}

type AmAmm struct {
	AmAmmManager common.Address `json:"am"`
	SwapFee0For1 *uint256.Int   `json:"sf01"`
	SwapFee1For0 *uint256.Int   `json:"sf10"`
}

type HookParams struct {
	FeeMin                     *uint256.Int `json:"fmin"`
	FeeMax                     *uint256.Int `json:"fmax"`
	FeeQuadraticMultiplier     *uint256.Int `json:"fqm"`
	FeeTwapSecondsAgo          uint32       `json:"ftsa"`
	SurgeFeeHalfLife           *uint256.Int `json:"sfhl"`
	SurgeFeeAutostartThreshold uint16       `json:"sfat"`
	VaultSurgeThreshold0       *uint256.Int `json:"vst0"`
	VaultSurgeThreshold1       *uint256.Int `json:"vst1"`
	AmAmmEnabled               bool         `json:"aae"`
	OracleMinInterval          uint32       `json:"omi"`
}

type Slot0 struct {
	SqrtPriceX96       *uint256.Int `json:"spx96"`
	Tick               int          `json:"t"`
	LastSwapTimestamp  uint32       `json:"lst"`
	LastSurgeTimestamp uint32       `json:"lsgt"`
}

type PoolState struct {
	Hooklet              common.Address `json:"h"`
	TwapSecondsAgo       uint32         `json:"tsa"`
	LdfParams            [32]byte       `json:"lp"`
	HookParams           []byte         `json:"hp"`
	LdfType              uint8          `json:"lt"`
	MinRawTokenRatio0    *uint256.Int   `json:"mrtr0"`
	TargetRawTokenRatio0 *uint256.Int   `json:"trtr0"`
	MaxRawTokenRatio0    *uint256.Int   `json:"xrtr0"`
	MinRawTokenRatio1    *uint256.Int   `json:"mrtr1"`
	TargetRawTokenRatio1 *uint256.Int   `json:"trtr1"`
	MaxRawTokenRatio1    *uint256.Int   `json:"xrtr1"`
	Currency0Decimals    uint8          `json:"c0d"`
	Currency1Decimals    uint8          `json:"c1d"`
	RawBalance0          *uint256.Int   `json:"rb0"`
	RawBalance1          *uint256.Int   `json:"rb1"`
	Reserve0             *uint256.Int   `json:"r0"`
	Reserve1             *uint256.Int   `json:"r1"`
	IdleBalance          [32]byte       `json:"ib"`
}

type Slot0RPC struct {
	SqrtPriceX96       *big.Int
	Tick               *big.Int
	LastSwapTimestamp  uint32
	LastSurgeTimestamp uint32
}

type BidRPC struct {
	Data struct {
		Manager  common.Address
		BlockIdx *big.Int
		Payload  [6]byte
		Rent     *big.Int
		Deposit  *big.Int
	}
}
type PoolStateRPC struct {
	Data struct {
		LiquidityDensityFunction common.Address
		BunniToken               common.Address
		Hooklet                  common.Address
		TwapSecondsAgo           *big.Int
		LdfParams                [32]byte
		HookParams               []byte
		Vault0                   common.Address
		Vault1                   common.Address
		LdfType                  uint8
		MinRawTokenRatio0        *big.Int
		TargetRawTokenRatio0     *big.Int
		MaxRawTokenRatio0        *big.Int
		MinRawTokenRatio1        *big.Int
		TargetRawTokenRatio1     *big.Int
		MaxRawTokenRatio1        *big.Int
		Currency0Decimals        uint8
		Currency1Decimals        uint8
		Vault0Decimals           uint8
		Vault1Decimals           uint8
		RawBalance0              *big.Int
		RawBalance1              *big.Int
		Reserve0                 *big.Int
		Reserve1                 *big.Int
		IdleBalance              [32]byte
	}
}

type LegacyPoolStateRPC struct {
	Data struct {
		LiquidityDensityFunction common.Address
		BunniToken               common.Address
		Hooklet                  common.Address
		TwapSecondsAgo           *big.Int
		LdfParams                [32]byte
		HookParams               []byte
		Vault0                   common.Address
		Vault1                   common.Address
		LdfType                  uint8
		MinRawTokenRatio0        *big.Int
		TargetRawTokenRatio0     *big.Int
		MaxRawTokenRatio0        *big.Int
		MinRawTokenRatio1        *big.Int
		TargetRawTokenRatio1     *big.Int
		MaxRawTokenRatio1        *big.Int
		RawBalance0              *big.Int
		RawBalance1              *big.Int
		Reserve0                 *big.Int
		Reserve1                 *big.Int
		IdleBalance              [32]byte
	}
}
