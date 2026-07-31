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
	TradeRate0             *uint256.Int   `json:"r0"`
	TradeRate1             *uint256.Int   `json:"r1"`
	PriceScale             *uint256.Int   `json:"ps"`
	WithdrawsQueued        *uint256.Int   `json:"wq"`
	WithdrawsClaimed       *uint256.Int   `json:"wc"`
	LiquidityAsset         common.Address `json:"la"`
	LiquidityAssetDecimals uint8          `json:"lad"`
	SwapTypes              SwapType       `json:"swapType"`
	ArmType                ArmType        `json:"armType"`
	HasWithdrawalQueue     bool           `json:"hasWithdrawalQueue"`
	Gas                    Gas            `json:"g"`
	// BaseAssets is only populated for ArmType Pricable4626. Tokens[0] is the liquidity asset and
	// BaseAssets[i] describes Tokens[i+1]. The upgraded ARM contract (AbstractARM) only allows swaps
	// between the liquidity asset and one of its base assets (star topology, no base-to-base swaps),
	// each priced and converted independently via baseAssetConfigs(baseAsset).
	BaseAssets []BaseAssetInfo `json:"bas"`
}

// BaseAssetInfo is the per-base-asset pricing/conversion snapshot for a Pricable4626 pool.
type BaseAssetInfo struct {
	Decimals               uint8        `json:"d"`
	PeggedToLiquidityAsset bool         `json:"pg"`
	BuyPrice               *uint256.Int `json:"bp"`
	SellPrice              *uint256.Int `json:"sp"`
	BuyLiquidityRemaining  *uint256.Int `json:"blr"`
	SellLiquidityRemaining *uint256.Int `json:"slr"`
	// ConvertRateAssetsPerShare/ConvertRateSharesPerAsset are only populated when
	// !PeggedToLiquidityAsset. They snapshot the base asset's adapter conversion
	// (adapter.convertToAssets(10^Decimals) / adapter.convertToShares(10^LiquidityAssetDecimals)) so the
	// (pure) simulator can apply a linear ratio at quote time instead of calling the adapter, which is
	// a (near-)linear share/asset converter (ERC4626 vault ratio, Lido shares, wstETH rate, etc).
	ConvertRateAssetsPerShare *uint256.Int `json:"cra"`
	ConvertRateSharesPerAsset *uint256.Int `json:"crs"`
}

type PoolState struct {
	Token0                 common.Address
	Token1                 common.Address
	TradeRate0             *big.Int
	TradeRate1             *big.Int
	PriceScale             *big.Int
	WithdrawsQueued        *big.Int
	WithdrawsClaimed       *big.Int
	Reserve0               *big.Int
	Reserve1               *big.Int
	LiquidityAsset         common.Address
	LiquidityAssetDecimals uint8
	// BaseAssets/BaseAssetReserves are only populated for ArmType Pricable4626, one entry per
	// getBaseAssets() address (same order); entity.Pool.Tokens ends up [Token0, BaseAssets[i].Address...].
	BaseAssets        []PoolStateBaseAsset
	BaseAssetReserves []*big.Int
}

// PoolStateBaseAsset is the *big.Int counterpart of BaseAssetInfo used while fetching on-chain state.
type PoolStateBaseAsset struct {
	Address                   common.Address
	Decimals                  uint8
	PeggedToLiquidityAsset    bool
	BuyPrice                  *big.Int
	SellPrice                 *big.Int
	BuyLiquidityRemaining     *big.Int
	SellLiquidityRemaining    *big.Int
	ConvertRateAssetsPerShare *big.Int
	ConvertRateSharesPerAsset *big.Int
}

// BaseAssetConfig mirrors AbstractARM's public baseAssetConfigs(address) getter. Different AbstractARM
// deployments return different tuple shapes (see baseAssetConfigsV2ABI); BaseAssetDecimals is only
// populated when the 9-field layout matches (fetchAssetAndState always fetches decimals separately via
// a plain ERC20 decimals() call too, so this field isn't relied upon).
type BaseAssetConfig struct {
	BuyPrice               *big.Int
	SellPrice              *big.Int
	BuyLiquidityRemaining  *big.Int
	SellLiquidityRemaining *big.Int
	CrossPrice             *big.Int
	PendingRedeemAssets    *big.Int
	PeggedToLiquidityAsset bool
	BaseAssetDecimals      uint8
	Adapter                common.Address
}

type Gas struct {
	ZeroToOne uint64 `json:"z2o,omitempty"`
	OneToZero uint64 `json:"o2z,omitempty"`
}
