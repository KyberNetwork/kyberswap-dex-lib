package genericarm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type SwapType uint8
type ArmType uint8

const (
	None SwapType = iota
	ZeroToOne
	OneToZero
	Both
)

const (
	Pegged ArmType = iota
	Pricable
	Pricable4626
)

type Extra struct {
	TradeRate0         *uint256.Int   `json:"r0"`
	TradeRate1         *uint256.Int   `json:"r1"`
	PriceScale         *uint256.Int   `json:"ps"`
	WithdrawsQueued    *uint256.Int   `json:"wq"`
	WithdrawsClaimed   *uint256.Int   `json:"wc"`
	LiquidityAsset     common.Address `json:"la"`
	SwapTypes          SwapType       `json:"swapType"`
	ArmType            ArmType        `json:"armType"`
	HasWithdrawalQueue bool           `json:"hasWithdrawalQueue"`
	Gas                Gas            `json:"g"`
	Vault              ERC4626Extra   `json:"v"`
}

type ERC4626Extra struct {
	BaseAsset   common.Address `json:"ba"`
	TotalAssets *uint256.Int   `json:"ta"`
	TotalSupply *uint256.Int   `json:"ts"`
	// BuyPrice/SellPrice are only populated for ArmType Pricable4626. The upgraded ARM contract
	// (AbstractARM) prices each base asset individually via baseAssetConfigs(baseAsset) instead of
	// the removed traderate0()/traderate1(); PRICE_SCALE is now a fixed 1e36 constant, not a getter.
	BuyPrice  *uint256.Int `json:"bp"`
	SellPrice *uint256.Int `json:"sp"`
}

type PoolState struct {
	Token0           common.Address
	Token1           common.Address
	TradeRate0       *big.Int
	TradeRate1       *big.Int
	PriceScale       *big.Int
	WithdrawsQueued  *big.Int
	WithdrawsClaimed *big.Int
	Reserve0         *big.Int
	Reserve1         *big.Int
	LiquidityAsset   common.Address
	Vault            ERC4626
}

type ERC4626 struct {
	BaseAsset   common.Address
	TotalAssets *big.Int
	TotalSupply *big.Int
	BuyPrice    *big.Int
	SellPrice   *big.Int
}

// BaseAssetConfig mirrors AbstractARM's public baseAssetConfigs(address) getter. Only BuyPrice/SellPrice
// are used today; the remaining fields are decoded to keep the ABI tuple unpacking correct.
type BaseAssetConfig struct {
	BuyPrice               *big.Int
	SellPrice              *big.Int
	BuyLiquidityRemaining  *big.Int
	SellLiquidityRemaining *big.Int
	CrossPrice             *big.Int
	PendingRedeemAssets    *big.Int
	PeggedToLiquidityAsset bool
	Adapter                common.Address
}
type Gas struct {
	ZeroToOne uint64 `json:"z2o,omitempty"`
	OneToZero uint64 `json:"o2z,omitempty"`
}
