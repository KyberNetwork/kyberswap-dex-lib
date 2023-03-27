package repository

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/usecase"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/usecase/types"
)

// IRPCRepository collects data from RPC nodes
type IRPCRepository interface {
	// GetLatestBlockTimestamp get the latest block timestamp
	GetLatestBlockTimestamp(ctx context.Context) (uint64, error)
	// Call executes one call to RPC nodes
	Call(ctx context.Context, in *CallParams) error
	// MultiCall executes multiple calls at once to RPC nodes
	MultiCall(ctx context.Context, calls []*CallParams) error
	// TryAggregate ...
	TryAggregate(ctx context.Context, requireSuccess bool, calls []*TryCallParams) error
	// TryAggregateForce ...
	TryAggregateForce(ctx context.Context, requireSuccess bool, calls []*TryCallParams) error
	// TryAggregateUnpack ...
	TryAggregateUnpack(ctx context.Context, requireSuccess bool, calls []*TryCallUnPackParams) error
}

// IPriceRepository collects, manage and store price data in datastore and cache
type IPriceRepository interface {
	IPriceDatastoreRepository
	IPriceCacheRepository

	// Save saves entity.Price in both datastore and cache
	Save(ctx context.Context, price entity.Price) error
}

// IPriceCacheRepository collects, manage and store price data in cache
type IPriceCacheRepository interface {
	// Keys returns all keys in cache
	Keys(ctx context.Context) []string
	// Get looks for entity.Price mapped with address in cache
	Get(ctx context.Context, address string) (entity.Price, error)
	// Set saves entity.Price mapped with address in cache
	Set(ctx context.Context, address string, price entity.Price) error
	// Remove removes price mapped with address from cache
	Remove(ctx context.Context, address string) error
	// Count return size of cache
	Count(ctx context.Context) int
}

// IPriceDatastoreRepository collects, manage and store price data in datastore
type IPriceDatastoreRepository interface {
	// FindAll returns all prices in datastore
	FindAll(ctx context.Context) ([]entity.Price, error)
	// FindByAddresses receives list of addresses and returns list of prices
	FindByAddresses(ctx context.Context, addresses []string) ([]entity.Price, error)
	// FindMapPriceByAddresses receives list of addresses and return a map with key is address and value is entity.Price
	FindMapPriceByAddresses(ctx context.Context, addresses []string) (map[string]float64, error)
	// Persist saves entity.Price to datastore
	Persist(ctx context.Context, price entity.Price) error
	// Delete physically delete entity.Price from datastore
	Delete(ctx context.Context, price entity.Price) error
	// DeleteMultiple delete multiple entity.Price from datastore
	DeleteMultiple(ctx context.Context, prices []entity.Price) error
}

// ITokenRepository collects, manage and store token data in datastore and cache
type ITokenRepository interface {
	ITokenDatastoreRepository
	ITokenCacheRepository

	// Save saves entity.Token in both datastore and cache
	Save(ctx context.Context, token entity.Token) error
}

// ITokenCacheRepository collects, manage and store token data in cache
type ITokenCacheRepository interface {
	// Keys returns all keys in cache
	Keys(ctx context.Context) []string
	// Get looks for entity.Token mapped with address in cache
	Get(ctx context.Context, address string) (entity.Token, error)
	// Set saves entity.Token mapped with address in cache
	Set(ctx context.Context, address string, token entity.Token) error
	// Remove removes token mapped with address from cache
	Remove(ctx context.Context, address string) error
	// Count return size of cache
	Count(ctx context.Context) int
	// GetByAddresses gets list of tokens based on a list of id
	GetByAddresses(ctx context.Context, ids []string) ([]entity.Token, error)
}

// ITokenDatastoreRepository collects, manage and store token data in datastore
type ITokenDatastoreRepository interface {
	// FindAll returns all tokens in datastore
	FindAll(ctx context.Context) ([]entity.Token, error)
	// FindByAddresses receives list of addresses and returns list of tokens
	FindByAddresses(ctx context.Context, addresses []string) ([]entity.Token, error)
	// Persist saves entity.Token to datastore
	Persist(ctx context.Context, token entity.Token) error
	// Delete physically delete entity.Token from datastore
	Delete(ctx context.Context, token entity.Token) error
}

// IPoolRepository collects, manage and store pool data in datastore and cache
type IPoolRepository interface {
	IPoolDatastoreRepository
	IPoolCacheRepository

	// Save saves entity.Pool in both datastore and cache
	Save(ctx context.Context, pool entity.Pool) error
}

// IPoolCacheRepository collects, manage and store pool data in cache
type IPoolCacheRepository interface {
	// Keys returns all keys in cache
	Keys(ctx context.Context) []string
	// Get looks for entity.Pool mapped with address in cache
	Get(ctx context.Context, address string) (entity.Pool, error)
	// GetByAddresses gets list of pools based on a list of id
	GetByAddresses(ctx context.Context, ids []string) ([]entity.Pool, error)
	// Set saves entity.Pool mapped with address in cache
	Set(ctx context.Context, address string, pool entity.Pool) error
	// Remove removes pool mapped with address from cache
	Remove(ctx context.Context, address string) error
	// Count return size of cache
	Count(ctx context.Context) int
	// GetPoolIdsByExchange gets list of pool ids belong to an exchange
	GetPoolIdsByExchange(ctx context.Context, id string) []string
	// GetPoolsByExchange gets list of pools belong to an exchange
	GetPoolsByExchange(ctx context.Context, id string) ([]entity.Pool, error)
	// IsPoolExist check if a pool exists in cache
	IsPoolExist(ctx context.Context, address string) bool
}

// IPoolDatastoreRepository collects, manage and store pool data in datastore
type IPoolDatastoreRepository interface {
	// FindAll returns all prices in datastore
	FindAll(ctx context.Context) ([]entity.Pool, error)
	// FindByAddresses receives list of addresses and returns list of pools
	FindByAddresses(ctx context.Context, addresses []string) ([]entity.Pool, error)
	// Persist saves entity.Pool to datastore
	Persist(ctx context.Context, pool entity.Pool) error
	// Delete delete entity.Pool from datastore
	Delete(ctx context.Context, pool entity.Pool) error
}

// IRouteRepository collects, manage and store route data in datastore
type IRouteRepository interface {
	GetBestPools(ctx context.Context, directPairKey, tokenIn, tokenOut string, opt usecase.GetBestPoolsOptions, whitelistI, whitelistJ bool) (*types.BestPools, error)
	AddToSortedSetScoreByReserveUsd(ctx context.Context, pool entity.Pool, key string, tokenIAddress, tokenJAddress string, whiteListI, whiteListJ bool) error
	AddToSortedSetScoreByAmplifiedTvl(ctx context.Context, pool entity.Pool, key string, tokenIAddress, tokenJAddress string, whiteListI, whiteListJ bool) error
}

type IStatsRepository interface {
	Get(ctx context.Context) (entity.Stats, error)
	Persist(ctx context.Context, stats entity.Stats) error
}

type IScannerStateRepository interface {
	GetDexOffset(ctx context.Context, offsetKey string) (int, error)
	SetDexOffset(ctx context.Context, offsetKey string, offset interface{}) error
	GetScanBlock(ctx context.Context) (uint64, error)
	SetScanBlock(ctx context.Context, block uint64) error
	GetGasPrice(ctx context.Context) (*big.Float, error)
	SetGasPrice(ctx context.Context, gasPrice string) error
	GetCurveAddressProviders(ctx context.Context) (string, error)
	SetCurveAddressProviders(ctx context.Context, providers string) error
}

// IRouteCacheRepository collects, manage and store route cache
type IRouteCacheRepository interface {
	// Set stores route cache
	Set(ctx context.Context, key string, data string, ttl time.Duration) error
	// Get receives key and return data, ttl and error if exists
	Get(ctx context.Context, key string) ([]byte, time.Duration, error)
}
