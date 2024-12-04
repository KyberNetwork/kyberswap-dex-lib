package fulcrom

import (
	"context"
	"math/big"
)

// IVaultReader reads vault smart contract
type IVaultReader interface {
	Read(ctx context.Context, address string) (*Vault, error)
}

// IVaultPriceFeedReader reads vault price feed smart contract
type IVaultPriceFeedReader interface {
	Read(ctx context.Context, address string, tokens []string) (*VaultPriceFeed, error)
}

// IUSDGReader reads usdg smart contract
type IUSDGReader interface {
	Read(ctx context.Context, address string) (*USDG, error)
}

type IFastPriceFeed interface {
	GetVersion() int
	GetPrice(token string, refPrice *big.Int, maximise bool) *big.Int
}
