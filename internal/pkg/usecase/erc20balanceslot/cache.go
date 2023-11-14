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
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type Cache struct {
	repo       erc20balanceslot.IRepository
	probe      *MultipleStrategy
	predefined map[common.Address]*entity.ERC20BalanceSlot
	cache      sync.Map // common.Address => *entity.ERC20BalanceSlot
	newEntries sync.Map // common.Address => *entity.ERC20BalanceSlot
	group      singleflight.Group
}

func NewCache(repo erc20balanceslot.IRepository, probe *MultipleStrategy, predefined map[string]*entity.ERC20BalanceSlot) *Cache {
	c := &Cache{
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
	if data, ok := preloadedByPrefix[c.repo.GetPrefix()]; ok {
		preloaded, err := DeserializePreloaded(data)
		if err != nil {
			return fmt.Errorf("could not deserialize preloaded ERC20 balance slots %w", err)
		}
		for token, bl := range preloaded {
			c.cache.Store(token, bl)
		}
		logger.Infof("preloaded %v token balance slots", len(preloaded))
	}
	// then from Redis
	entries, err := c.repo.GetAll(ctx)
	if err != nil {
		return err
	}
	for token, entry := range entries {
		c.cache.Store(token, entry)
	}
	logger.Infof("preloaded %v token balance slots from Redis", len(entries))
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
	// try to (re)probe the token
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
		// store the result
		if bl != nil {
			c.cache.Store(token, bl)
			c.newEntries.Store(token, bl)
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

// CommitToRedis writes all new entries to Redis.
func (c *Cache) CommitToRedis(ctx context.Context) (int, error) {
	logger.Infof("committing newly probed balance slots to Redis")
	var newEntries []*entity.ERC20BalanceSlot
	c.newEntries.Range(func(_, value any) bool {
		bl := value.(*entity.ERC20BalanceSlot)
		newEntries = append(newEntries, bl)
		return true
	})
	if err := c.repo.PutMany(ctx, newEntries); err != nil {
		return 0, err
	}
	for _, bl := range newEntries {
		c.newEntries.Delete(common.HexToAddress(bl.Token))
	}
	logger.Infof("committed %d balance slots to Redis", len(newEntries))
	return len(newEntries), nil
}
