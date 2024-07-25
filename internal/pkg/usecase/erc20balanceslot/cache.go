package erc20balanceslot

import (
	"context"
	"fmt"
	"sync"

	dexentity "github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/sync/singleflight"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/erc20balanceslot"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type Cache struct {
	chainID    valueobject.ChainID
	repo       erc20balanceslot.IRepository
	probe      *MultipleStrategy
	predefined map[common.Address]*entity.ERC20BalanceSlot
	cache      sync.Map // common.Address => *entity.ERC20BalanceSlot
	group      singleflight.Group
}

func NewCache(repo erc20balanceslot.IRepository, probe *MultipleStrategy, predefined map[string]*entity.ERC20BalanceSlot, chainID valueobject.ChainID) *Cache {
	c := &Cache{
		chainID:    chainID,
		repo:       repo,
		probe:      probe,
		predefined: make(map[common.Address]*entity.ERC20BalanceSlot),
	}
	for token, bl := range predefined {
		c.predefined[common.HexToAddress(token)] = bl
	}

	return c
}

func (c *Cache) PreloadAll(ctx context.Context) error {
	// preload from preloaded first
	if data, ok := preloadedByPrefix[c.chainID]; ok {
		preloaded, err := DeserializePreloaded(data)
		if err != nil {
			return fmt.Errorf("could not deserialize preloaded ERC20 balance slots %w", err)
		}
		for token, bl := range preloaded {
			c.cache.Store(token, bl)
		}
		logger.Debugf(ctx, "preloaded %v token balance slots", len(preloaded))
	}
	// then from Redis
	entries, err := c.repo.GetAll(ctx)
	if err != nil {
		return err
	}
	var numRedisPreloaded int
	for token, entry := range entries {
		if _oldEntry, ok := c.cache.Load(token); ok {
			oldEntry := _oldEntry.(*entity.ERC20BalanceSlot)
			if oldEntry.Found && !entry.Found {
				// only skip if old entry is found but new entry is not found
				continue
			}
		}
		c.cache.Store(token, entry)
		numRedisPreloaded++
	}
	logger.Debugf(ctx, "preloaded %v token balance slots from Redis", numRedisPreloaded)
	return nil
}

func (c *Cache) Get(ctx context.Context, token common.Address, pool *dexentity.Pool) (*entity.ERC20BalanceSlot, error) {
	// try predefined first
	if entry, ok := c.predefined[token]; ok {
		return entry, nil
	}
	// then try cache
	if _entry, ok := c.cache.Load(token); ok {
		entry := _entry.(*entity.ERC20BalanceSlot)
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
		var oldBl *entity.ERC20BalanceSlot
		if _oldBl, ok := c.cache.Load(token); ok {
			oldBl = _oldBl.(*entity.ERC20BalanceSlot)
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
	return _bl.(*entity.ERC20BalanceSlot), nil
}
