package erc20balanceslot

import (
	"context"
	"errors"
	"math/rand"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis"
	"github.com/ethereum/go-ethereum/common"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"github.com/KyberNetwork/router-service/internal/pkg/repository/erc20balanceslot"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

const (
	btcbAddr  = "0x152b9d0fdc40c096757f570a51e494bd4b943e50"
	wetheAddr = "0x49d5c2bdffac6ce2bfdb6640f4f80f226bc10bab"
)

type panicIfProbeMoreThanOnce struct {
	probe  IProbe
	probed sync.Map // common.Address => struct{}
}

func (p *panicIfProbeMoreThanOnce) GetWallet() common.Address {
	return p.probe.GetWallet()
}

func (p *panicIfProbeMoreThanOnce) ProbeBalanceSlot(token common.Address) (common.Hash, error) {
	_, probed := p.probed.Load(token)
	if probed {
		panic("only probe once")
	}
	result, err := p.probe.ProbeBalanceSlot(token)
	_, probed = p.probed.Load(token)
	if probed {
		panic("only probe once")
	}
	p.probed.Store(token, struct{}{})
	return result, err
}

type testProbe struct{}

func (*testProbe) GetWallet() common.Address {
	return common.Address{}
}

func (*testProbe) ProbeBalanceSlot(token common.Address) (common.Hash, error) {
	m := map[common.Address]common.Hash{
		common.HexToAddress(btcbAddr):  common.HexToHash("0x4f1749155d837e5f5ef076382254c01af904c6ddb97b100fef402248f448ea99"),
		common.HexToAddress(wetheAddr): common.HexToHash("0x4f1749155d837e5f5ef076382254c01af904c6ddb97b100fef402248f448ea99"),
	}
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
	slot, ok := m[token]
	if !ok {
		return common.Hash{}, errors.New("not found")
	}
	return slot, nil
}

const (
	prefix = "avalanche"
)

func TestPreload(t *testing.T) {
	logger.SetLogLevel("info")

	rd, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer rd.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: rd.Addr()})

	repo := erc20balanceslot.NewRedisRepository(redisClient, erc20balanceslot.RedisRepositoryConfig{
		Prefix: prefix,
	})
	c := NewCache(repo, &testProbe{}, nil)
	require.NoError(t, c.PreloadAll(context.Background()))
}

func TestGetBalanceSlot(t *testing.T) {
	logger.SetLogLevel("info")

	rd, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer rd.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: rd.Addr()})

	repo := erc20balanceslot.NewRedisRepository(redisClient, erc20balanceslot.RedisRepositoryConfig{
		Prefix: prefix,
	})
	c := NewCache(repo, &testProbe{}, nil)

	bl, err := c.Get(context.Background(), common.HexToAddress(btcbAddr))
	require.NoError(t, err)
	require.NotEmptyf(t, bl.BalanceSlot, "must have balance slot")

	// set rpcClient to nil to test if the subsequent request uses cached balance slot
	origProbe := c.probe
	c.probe = nil
	bl, err = c.Get(context.Background(), common.HexToAddress(btcbAddr))
	require.NoError(t, err)
	require.NotEmptyf(t, bl.BalanceSlot, "must have cached balance slot")

	// if a token is probed more than once concurently, the test will panic
	c.probe = &panicIfProbeMoreThanOnce{probe: origProbe}

	var wg errgroup.Group
	for i := 0; i < 100; i++ {
		wg.Go(func() error {
			_, err := c.Get(context.Background(), common.HexToAddress(wetheAddr))
			return err
		})
	}
	require.NoError(t, wg.Wait())

	// must commit newly probed token to redis
	numCommit, err := c.CommitToRedis(context.Background())
	require.NoError(t, err)
	require.Equal(t, numCommit, 2)

	bls, err := redisClient.HGetAll(context.Background(), utils.Join(prefix, erc20balanceslot.KeyERC20BalanceSlot)).Result()
	require.NoError(t, err)
	require.Truef(t, len(bls) == 2, "there must be 2 balance slots")
	for _, token := range []string{btcbAddr, wetheAddr} {
		_, ok := bls[strings.ToLower(token)]
		require.Truef(t, ok, "must have token %s", token)
	}

	// subsequent commit should commit nothing
	numCommit, err = c.CommitToRedis(context.Background())
	require.NoError(t, err)
	require.Equalf(t, numCommit, 0, "there must nothing to commit")
}
