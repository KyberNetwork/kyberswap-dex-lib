package synthetix

import "context"

// IPoolStateReader reads synthetix smart contract
type IPoolStateReader interface {
	Read(ctx context.Context, address string) (*PoolState, error)
}

// ISystemSettingsReader reads SystemSettings smart contract
type ISystemSettingsReader interface {
	Read(ctx context.Context, poolState *PoolState) (*SystemSettings, error)
}

// IExchangerWithFeeRecAlternativesReader reads ExchangerWithFeeRecAlternatives smart contract
type IExchangerWithFeeRecAlternativesReader interface {
	Read(ctx context.Context, poolState *PoolState) (*PoolState, error)
}

// IExchangeRatesReader reads ExchangeRates smart contract
type IExchangeRatesReader interface {
	Read(ctx context.Context, poolState *PoolState) (*PoolState, error)
}

// IChainlinkDataFeedReader reads Chainlink data feed smart contract
type IChainlinkDataFeedReader interface {
	Read(ctx context.Context, address string, roundCount int) (*ChainlinkDataFeed, error)
}

// IDexPriceAggregatorUniswapV3Reader reads DexPriceAggregatorUniswapV3 data feed smart contract
type IDexPriceAggregatorUniswapV3Reader interface {
	Read(ctx context.Context, poolState *PoolState) (*DexPriceAggregatorUniswapV3, error)
}
