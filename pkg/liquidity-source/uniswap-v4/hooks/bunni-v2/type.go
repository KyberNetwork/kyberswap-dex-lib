package bunniv2

import (
	"math/big"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/hooklet"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/ldf"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/oracle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type Hook struct {
	uniswapv4.Hook
	HookExtra
	hook common.Address

	ldf         ldf.ILiquidityDensityFunction
	oracle      *oracle.ObservationStorage
	hooklet     hooklet.IHooklet
	isNative    [2]bool
	tickSpacing int

	// rebalanceOrderDeadline *uint256.Int
}

type HookExtra struct {
	HookletExtra        string
	HookletAddress      common.Address
	LDFAddress          common.Address
	HookFee             *uint256.Int
	PoolManagerReserves [2]*uint256.Int
	LdfState            [32]byte
	Vaults              [2]Vault
	AmAmm               AmAmm
	ObservationState    ObservationState
	CuratorFees         CuratorFees
	Observations        []*oracle.Observation
	HookParams          HookParams
	Slot0               Slot0
	BunniState          PoolState
	VaultSharePrices    VaultSharePrices
}

type LdfState struct {
	Initialized bool
	LastMinTick int
}

type Vault struct {
	Address    common.Address
	Decimals   uint8
	RedeemRate *uint256.Int
	MaxDeposit *uint256.Int
}
type ObservationState struct {
	Index                   uint32
	Cardinality             uint32
	CardinalityNext         uint32
	IntermediateObservation *oracle.Observation
}

type VaultSharePrices struct {
	Initialized  bool
	SharedPrice0 *uint256.Int
	SharedPrice1 *uint256.Int
}

type CuratorFees struct {
	FeeRate *uint256.Int
}

type AmAmm struct {
	AmAmmManager common.Address
	SwapFee0For1 *uint256.Int
	SwapFee1For0 *uint256.Int
}

type HookParams struct {
	FeeMin                 *uint256.Int
	FeeMax                 *uint256.Int
	FeeQuadraticMultiplier *uint256.Int
	FeeTwapSecondsAgo      uint32
	// MaxAmAmmFee                *uint256.Int
	SurgeFeeHalfLife           *uint256.Int
	SurgeFeeAutostartThreshold uint16
	VaultSurgeThreshold0       *uint256.Int
	VaultSurgeThreshold1       *uint256.Int
	// RebalanceThreshold         uint16
	// RebalanceMaxSlippage       uint16
	// RebalanceTwapSecondsAgo    uint16
	// RebalanceOrderTTL          uint16
	AmAmmEnabled      bool
	OracleMinInterval uint32
	// MinRentMultiplier          *uint256.Int
}

type Slot0 struct {
	SqrtPriceX96       *uint256.Int
	Tick               int
	LastSwapTimestamp  uint32
	LastSurgeTimestamp uint32
}

type PoolState struct {
	// LiquidityDensityFunction common.Address
	// BunniToken           common.Address
	Hooklet              common.Address
	TwapSecondsAgo       uint32
	LdfParams            [32]byte
	HookParams           []byte
	LdfType              uint8
	MinRawTokenRatio0    *uint256.Int
	TargetRawTokenRatio0 *uint256.Int
	MaxRawTokenRatio0    *uint256.Int
	MinRawTokenRatio1    *uint256.Int
	TargetRawTokenRatio1 *uint256.Int
	MaxRawTokenRatio1    *uint256.Int
	Currency0Decimals    uint8
	Currency1Decimals    uint8
	RawBalance0          *uint256.Int
	RawBalance1          *uint256.Int
	Reserve0             *uint256.Int
	Reserve1             *uint256.Int
	IdleBalance          [32]byte
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
