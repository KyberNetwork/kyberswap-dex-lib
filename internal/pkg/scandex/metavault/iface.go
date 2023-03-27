package metavault

// NOTE: we generate mock files in the same package to avoid cycle dependencies
//go:generate mockgen -destination ./mock_vault_reader.go -package metavault dmm-aggregator-backend/internal/pkg/scandex/metavault IVaultReader
//go:generate mockgen -destination ./mock_vault_price_feed_reader.go -package metavault dmm-aggregator-backend/internal/pkg/scandex/metavault IVaultPriceFeedReader
//go:generate mockgen -destination ./mock_fast_price_feed_reader_v1.go -package metavault dmm-aggregator-backend/internal/pkg/scandex/metavault IFastPriceFeedV1Reader
//go:generate mockgen -destination ./mock_price_feed_reader.go -package metavault dmm-aggregator-backend/internal/pkg/scandex/metavault IPriceFeedReader
//go:generate mockgen -destination ./mock_usdm_reader.go -package metavault dmm-aggregator-backend/internal/pkg/scandex/metavault IUSDMReader
//go:generate mockgen -destination ./mock_chainlink_flags_reader.go -package metavault dmm-aggregator-backend/internal/pkg/scandex/metavault IChainlinkFlagsReader
//go:generate mockgen -destination ./mock_pancake_pair_reader.go -package metavault dmm-aggregator-backend/internal/pkg/scandex/metavault IPancakePairReader

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
}
