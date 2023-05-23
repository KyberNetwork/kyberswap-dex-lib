package metavault

// NOTE: we generate mock files in the same package to avoid cycle dependencies
//go:generate mockgen -destination ./mock_vault_reader.go -package metavault pool-service/internal/dex/metavault IVaultReader
//go:generate mockgen -destination ./mock_vault_price_feed_reader.go -package metavault pool-service/internal/dex/metavault IVaultPriceFeedReader
//go:generate mockgen -destination ./mock_fast_price_feed_reader_v1.go -package metavault pool-service/internal/dex/metavault IFastPriceFeedV1Reader
//go:generate mockgen -destination ./mock_price_feed_reader.go -package metavault pool-service/internal/dex/metavault IPriceFeedReader
//go:generate mockgen -destination ./mock_usdm_reader.go -package metavault pool-service/internal/dex/metavault IUSDMReader

import "context"

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
}
