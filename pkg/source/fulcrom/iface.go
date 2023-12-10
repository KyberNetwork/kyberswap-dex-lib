package fulcrom

// NOTE: we generate mock files in the same package to avoid cycle dependencies
//go:generate mockgen -destination ./mock_vault_reader.go -package gmx pool-service/internal/dex/gmx IVaultReader
//go:generate mockgen -destination ./mock_vault_price_feed_reader.go -package gmx pool-service/internal/dex/gmx IVaultPriceFeedReader
//go:generate mockgen -destination ./mock_fast_price_feed_reader_v1.go -package gmx pool-service/internal/dex/gmx IFastPriceFeedV1Reader
//go:generate mockgen -destination ./mock_price_feed_reader.go -package gmx pool-service/internal/dex/gmx IPriceFeedReader
//go:generate mockgen -destination ./mock_usdg_reader.go -package gmx pool-service/internal/dex/gmx IUSDGReader
//go:generate mockgen -destination ./mock_chainlink_flags_reader.go -package gmx pool-service/internal/dex/gmx IChainlinkFlagsReader
//go:generate mockgen -destination ./mock_pancake_pair_reader.go -package gmx pool-service/internal/dex/gmx IPancakePairReader

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
