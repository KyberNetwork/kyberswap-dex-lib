package gmx

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

// IUSDRReader reads usdr smart contract
type IUSDRReader interface {
	Read(ctx context.Context, address string) (*USDR, error)
}

// IChainlinkFlagsReader reads chainlink flag smart contract
type IChainlinkFlagsReader interface {
	Read(ctx context.Context, address string) (*ChainlinkFlags, error)
}

// IPancakePairReader reads pancake pair smart contract
type IPancakePairReader interface {
	Read(ctx context.Context, address string) (*PancakePair, error)
}

type IFastPriceFeed interface {
	GetVersion() int
	GetPrice(token string, refPrice *big.Int, maximise bool) *big.Int
}
