package erc20balanceslot

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/sync/singleflight"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/erc20balanceslot"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type Cache struct {
	repo       erc20balanceslot.IRepository
	probe      IProbe
	predefined map[common.Address]*entity.ERC20BalanceSlot
	cache      sync.Map // common.Address => *entity.ERC20BalanceSlot
	newEntries sync.Map // common.Address => *entity.ERC20BalanceSlot
	group      singleflight.Group
}

func NewCache(repo erc20balanceslot.IRepository, probe IProbe, predefined map[string]*entity.ERC20BalanceSlot) *Cache {
	c := &Cache{
		repo:       repo,
		probe:      probe,
		predefined: make(map[common.Address]*entity.ERC20BalanceSlot),
	}
	for token, bl := range predefined {
		c.predefined[common.HexToAddress(token)] = bl
	}

	// Set up channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.
	sigCh := make(chan os.Signal, 1)

	// Passing no signals to Notify means that
	// all signals will be sent to the channel.
	signal.Notify(sigCh)

	go func() {
		s := <-sigCh
		logger.Infof("got signal %s, commit to Redis", s)
		c.CommitToRedis(context.Background())
	}()

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

func (c *Cache) Get(ctx context.Context, token common.Address) (*entity.ERC20BalanceSlot, error) {
	// try predefined first
	if entry, ok := c.predefined[token]; ok {
		return entry, nil
	}
	// then try cache
	if entry, ok := c.cache.Load(token); ok {
		return entry.(*entity.ERC20BalanceSlot), nil
	}
	// then probe it
	_slot, err, _ := c.group.Do(token.String(), func() (interface{}, error) {
		return c.probe.ProbeBalanceSlot(token)
	})
	if err != nil {
		return nil, err
	}
	slot := _slot.(common.Hash)
	bl := &entity.ERC20BalanceSlot{
		Token:       strings.ToLower(token.String()),
		Wallet:      strings.ToLower(c.probe.GetWallet().String()),
		Found:       true,
		BalanceSlot: slot.Hex(),
	}
	c.cache.Store(token, bl)
	c.newEntries.Store(token, bl)
	return bl, nil
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
