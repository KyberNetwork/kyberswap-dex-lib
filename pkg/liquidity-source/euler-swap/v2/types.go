package v2

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/euler-swap/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type StaticExtra struct {
	SupplyVault0 string `json:"sv0"`
	SupplyVault1 string `json:"sv1"`
	BorrowVault0 string `json:"bv0,omitempty"`
	BorrowVault1 string `json:"bv1,omitempty"`
	EulerAccount string `json:"ea"`
	FeeRecipient string `json:"fr,omitempty"`
	EVC          string `json:"evc"`
}

type DynamicParams struct {
	EquilibriumReserve0 *uint256.Int `json:"er0"`
	EquilibriumReserve1 *uint256.Int `json:"er1"`
	MinReserve0         *uint256.Int `json:"mr0"`
	MinReserve1         *uint256.Int `json:"mr1"`
	PriceX              *uint256.Int `json:"px"`
	PriceY              *uint256.Int `json:"py"`
	ConcentrationX      *uint256.Int `json:"cx"`
	ConcentrationY      *uint256.Int `json:"cy"`
	Fee0                *uint256.Int `json:"f0"`
	Fee1                *uint256.Int `json:"f1"`
	Expiration          uint64       `json:"exp,omitempty"`
	SwapHookedOps       uint8        `json:"sho,omitempty"`
	SwapHook            string       `json:"sh,omitempty"`
}

type Extra struct {
	DynamicParams
	Pause           uint32                `json:"p,omitempty"` // 0 = unactivated, 1 = unlocked, 2 = locked
	SupplyVault     [2]*shared.VaultState `json:"sv"`          // supply vault states
	BorrowVault     [3]*shared.VaultState `json:"bv"`          // borrow vault states (can be nil)
	ControllerVault string                `json:"cV,omitempty"`
	Collaterals     []*uint256.Int        `json:"c,omitempty"` // collateral amounts
	HookExtra       string                `json:"he,omitempty"`
}

type PoolExtra struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type AssetsRPC struct {
	Asset0 common.Address
	Asset1 common.Address
}

// StaticParamsFields mirrors IEulerSwap.StaticParams. It is also the shape of
// the "sParams" tuple carried inline by the registry's PoolRegistered event,
// so it's reused for decoding that event without an extra RPC round trip.
type StaticParamsFields struct {
	SupplyVault0 common.Address `abi:"supplyVault0"`
	SupplyVault1 common.Address `abi:"supplyVault1"`
	BorrowVault0 common.Address `abi:"borrowVault0"`
	BorrowVault1 common.Address `abi:"borrowVault1"`
	EulerAccount common.Address `abi:"eulerAccount"`
	FeeRecipient common.Address `abi:"feeRecipient"`
}

type StaticParamsRPC struct {
	Data StaticParamsFields
}

type DynamicParamsFields struct {
	EquilibriumReserve0  *big.Int       `abi:"equilibriumReserve0"`
	EquilibriumReserve1  *big.Int       `abi:"equilibriumReserve1"`
	MinReserve0          *big.Int       `abi:"minReserve0"`
	MinReserve1          *big.Int       `abi:"minReserve1"`
	PriceX               *big.Int       `abi:"priceX"`
	PriceY               *big.Int       `abi:"priceY"`
	ConcentrationX       uint64         `abi:"concentrationX"`
	ConcentrationY       uint64         `abi:"concentrationY"`
	Fee0                 uint64         `abi:"fee0"`
	Fee1                 uint64         `abi:"fee1"`
	Expiration           *big.Int       `abi:"expiration"`
	SwapHookedOperations uint8          `abi:"swapHookedOperations"`
	SwapHook             common.Address `abi:"swapHook"`
}

type DynamicParamsRPC struct {
	Data DynamicParamsFields
}

// buildStaticExtra maps a pool's static params (from RPC or the
// PoolRegistered event log, both share this shape) plus its EVC address into
// the persisted StaticExtra.
func buildStaticExtra(sp StaticParamsFields, evc common.Address) StaticExtra {
	staticExtra := StaticExtra{
		SupplyVault0: sp.SupplyVault0.Hex(),
		SupplyVault1: sp.SupplyVault1.Hex(),
		EulerAccount: sp.EulerAccount.Hex(),
		EVC:          evc.Hex(),
	}

	if !valueobject.IsZeroAddress(sp.BorrowVault0) {
		staticExtra.BorrowVault0 = sp.BorrowVault0.Hex()
	}
	if !valueobject.IsZeroAddress(sp.BorrowVault1) {
		staticExtra.BorrowVault1 = sp.BorrowVault1.Hex()
	}
	if !valueobject.IsZeroAddress(sp.FeeRecipient) {
		staticExtra.FeeRecipient = sp.FeeRecipient.Hex()
	}

	return staticExtra
}

// buildDynamicParams maps a pool's dynamic params fetched via RPC into the
// persisted DynamicParams.
func buildDynamicParams(dp DynamicParamsFields) DynamicParams {
	return DynamicParams{
		EquilibriumReserve0: uint256.MustFromBig(dp.EquilibriumReserve0),
		EquilibriumReserve1: uint256.MustFromBig(dp.EquilibriumReserve1),
		MinReserve0:         uint256.MustFromBig(dp.MinReserve0),
		MinReserve1:         uint256.MustFromBig(dp.MinReserve1),
		PriceX:              uint256.MustFromBig(dp.PriceX),
		PriceY:              uint256.MustFromBig(dp.PriceY),
		ConcentrationX:      uint256.NewInt(dp.ConcentrationX),
		ConcentrationY:      uint256.NewInt(dp.ConcentrationY),
		Fee0:                uint256.NewInt(dp.Fee0),
		Fee1:                uint256.NewInt(dp.Fee1),
		Expiration:          dp.Expiration.Uint64(),
		SwapHookedOps:       dp.SwapHookedOperations,
		SwapHook:            dp.SwapHook.Hex(),
	}
}

type TrackerData struct {
	Vaults               []shared.VaultRPC
	Reserves             shared.ReserveRPC
	DynamicParams        DynamicParamsRPC
	Controller           string          // controller debt vault, if exist
	VaultPrices          [][][2]*big.Int // other vault -> debt vault -> [bid/value,ask/debt]
	VaultLtvs            [][]uint16      // vault 0/1/controller -> debt vault
	CollatAmounts        []*big.Int      // asset amount of euler account across collateral vaults
	CollatPrices         [][][2]*big.Int // collat -> debt vault -> [bid,ask]
	CollatLtvs           [][]uint16      // collat -> debt vault
	IsOperatorAuthorized bool
	UniqueVaultAddresses []string // addresses corresponding to Vaults
}
