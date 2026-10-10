package apefun

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// policyRaw decodes LaunchFeeProtectionHook.policies(poolId), field order matching hookABI's outputs. Destination
// and Distributor only say whether the pool is registered (destination != 0); pricing reads Tax and Launched.
type policyRaw struct {
	Destination common.Address
	Distributor common.Address
	Tax         uint16
	Launched    uint64
}

// poolKeyRaw and launchRaw decode LaunchRegistryFactory.launchForToken(token), the Launch struct in full (a static
// tuple decodes only with every component). Pricing reads Stage and PoolId; the rest keeps the layout.
type poolKeyRaw struct {
	Currency0   common.Address
	Currency1   common.Address
	Fee         *big.Int
	TickSpacing *big.Int
	Hooks       common.Address
}

// launchResult is the decode target: go-ethereum unpacks a single tuple output into the first field of a struct.
type launchResult struct{ Launch launchRaw }

type launchRaw struct {
	Token            common.Address
	PoolId           [32]byte
	Key              poolKeyRaw
	PositionSalt     [32]byte
	Creator          common.Address
	FeeDestination   common.Address
	Distributor      common.Address
	CreatorTaxBps    uint16
	LaunchTime       uint64
	OpeningTick      *big.Int
	LowerTick        *big.Int
	UpperTick        *big.Int
	InitialLiquidity *big.Int
	MintedInventory  *big.Int
	MintDust         *big.Int
	MetadataHash     [32]byte
	Stage            uint8
	BuybackVault     common.Address
}
