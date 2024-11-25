package erc20balanceslot

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/types"
	dexentity "github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/sync/singleflight"

	"github.com/KyberNetwork/router-service/internal/pkg/repository/erc20balanceslot"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type Cache struct {
	chainID    valueobject.ChainID
	repo       erc20balanceslot.IRepository
	probe      *MultipleStrategy
	predefined map[common.Address]*types.ERC20BalanceSlot
	cache      sync.Map // common.Address => *types.ERC20BalanceSlot
	group      singleflight.Group

	preloadAttempted mapset.Set[common.Address]
}

func NewCache(repo erc20balanceslot.IRepository, probe *MultipleStrategy, predefined map[string]*types.ERC20BalanceSlot, chainID valueobject.ChainID) *Cache {
	c := &Cache{
		chainID:          chainID,
		repo:             repo,
		probe:            probe,
		predefined:       make(map[common.Address]*types.ERC20BalanceSlot),
		preloadAttempted: mapset.NewSet[common.Address](),
	}
	for token, bl := range predefined {
		c.predefined[common.HexToAddress(token)] = bl
	}

	return c
}

func (c *Cache) PreloadMany(ctx context.Context, tokens []common.Address) error {
	var (
		numInserted int
		start       = time.Now()
	)
	defer func() {
		logger.Debugf(ctx, "preloaded %v token balance slots from Redis took %s", numInserted, time.Since(start))
	}()

	var preloadingTokens []common.Address
	for _, token := range tokens {
		if !c.preloadAttempted.ContainsOne(token) {
			preloadingTokens = append(preloadingTokens, token)
		}
	}
	// logger.Debugf(ctx, "diffing took %s", time.Since(start))

	if len(preloadingTokens) == 0 {
		return nil
	}

	redisCount, err := c.repo.Count(ctx)
	if err != nil {
		return fmt.Errorf("Count returns error: %w", err)
	}

	var entries map[common.Address]*types.ERC20BalanceSlot
	// If the number of ERC20 balance slots in Redis is less than the number of querying addresses,
	// it's might faster to GetAll from Redis instead of GetMany.
	if redisCount < len(preloadingTokens) {
		entries, err = c.repo.GetAll(ctx)
		if err != nil {
			return fmt.Errorf("GetAll returns error: %w", err)
		}
		// remove unrelated entries
		tokensSet := mapset.NewThreadUnsafeSet(preloadingTokens...)
		for token := range entries {
			if !tokensSet.ContainsOne(token) {
				delete(entries, token)
			}
		}
	} else {
		entries, err = c.repo.GetMany(ctx, preloadingTokens)
		if err != nil {
			return fmt.Errorf("GetMany returns error: %w", err)
		}
	}
	for token, entry := range entries {
		if _oldEntry, ok := c.cache.Load(token); ok {
			oldEntry := _oldEntry.(*types.ERC20BalanceSlot)
			if oldEntry.Found && !entry.Found {
				// only skip if old entry is found but new entry is not found
				continue
			}
		}
		c.cache.Store(token, entry)
		numInserted++
	}

	c.preloadAttempted.Append(preloadingTokens...)

	return nil
}

func (c *Cache) PreloadFromEmbedded(ctx context.Context) error {
	start := time.Now()
	var numPreloaded int
	if data, ok := embeddedByPrefix[c.chainID]; ok {
		embedded, err := DeserializeEmbedded(data)
		if err != nil {
			return fmt.Errorf("could not deserialize embedded ERC20 balance slots %w", err)
		}
		for token, bl := range embedded {
			c.cache.Store(token, bl)
			c.preloadAttempted.Add(token)
		}
		numPreloaded = len(embedded)
	}
	logger.Debugf(ctx, "preloaded %v token balance slots from embedded took %s", numPreloaded, time.Since(start))
	return nil
}

func (c *Cache) Get(ctx context.Context, token common.Address, pool *dexentity.Pool) (*types.ERC20BalanceSlot, error) {
	// try predefined first
	if entry, ok := c.predefined[token]; ok {
		return entry, nil
	}
	// then try cache
	if _entry, ok := c.cache.Load(token); ok {
		entry := _entry.(*types.ERC20BalanceSlot)
		if entry.Found {
			return entry, nil
		}
	}
	// then try Redis
	if entry, _ := c.repo.Get(ctx, token); entry != nil {
		if entry.Found {
			c.cache.Store(token, entry)
			return entry, nil
		}
	}
	// then try to (re)probe the token
	_bl, err, _ := c.group.Do(token.String(), func() (interface{}, error) {
		var oldBl *types.ERC20BalanceSlot
		if _oldBl, ok := c.cache.Load(token); ok {
			oldBl = _oldBl.(*types.ERC20BalanceSlot)
		}
		extraParams := &MultipleStrategyExtraParams{}
		// only use addressable pool
		if pool != nil && common.IsHexAddress(pool.Address) {
			extraParams.DoubleFromSource = &DoubleFromSourceStrategyExtraParams{
				Source: common.HexToAddress(pool.Address),
			}
		}
		bl, err := c.probe.ProbeBalanceSlot(ctx, token, oldBl, extraParams)
		// store the result to cache and Redis
		if bl != nil {
			c.cache.Store(token, bl)
			if err := c.repo.Put(ctx, bl); err != nil {
				logger.WithFields(ctx, logger.Fields{"entity": bl}).Errorf("could not store balance slot: %s", err)
			}
		}
		// err != nil implies bl == nil
		if err != nil {
			return nil, fmt.Errorf("could not find balance slots: %w", err)
		}
		if !bl.Found {
			return nil, fmt.Errorf("could not find balance slots, attempted: %v", bl.StrategiesAttempted)
		}
		return bl, nil
	})
	if err != nil {
		return nil, err
	}
	return _bl.(*types.ERC20BalanceSlot), nil
}
