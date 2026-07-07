package metavault

// NOTE: we generate mock files in the same package to avoid cycle dependencies
//go:generate go run go.uber.org/mock/mockgen -destination ./mocks.go -package metavault . IVaultReader,IVaultPriceFeedReader,IFastPriceFeedV1Reader,IPriceFeedReader,IUSDMReader

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

// IFastPriceFeedV1Reader reads fast price feed smart contract
type IFastPriceFeedV1Reader interface {
	Read(ctx context.Context, address string, tokens []string) (*FastPriceFeedV1, error)
}

type IFastPriceFeedV2Reader interface {
	Read(ctx context.Context, address string, tokens []string) (*FastPriceFeedV2, error)
}

// IPriceFeedReader reads price feed smart contract
type IPriceFeedReader interface {
	Read(ctx context.Context, address string, roundCount int) (*PriceFeed, error)
}

// IUSDMReader reads usdm smart contract
type IUSDMReader interface {
	Read(ctx context.Context, address string) (*USDM, error)
}

type IFastPriceFeed interface {
	GetVersion() int
	GetPrice(token string, refPrice *big.Int, maximise bool) *big.Int
}
