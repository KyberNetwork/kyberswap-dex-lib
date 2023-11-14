package erc20balanceslot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis"
	"github.com/ethereum/go-ethereum/common"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/erc20balanceslot"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

const (
	btcbAddr  = "0x152b9d0fdc40c096757f570a51e494bd4b943e50"
	wetheAddr = "0x49d5c2bdffac6ce2bfdb6640f4f80f226bc10bab"
)

type testProbe struct{}

func (*testProbe) Name(_ ProbeStrategyExtraParams) string {
	return "test_probe"
}

func (*testProbe) ProbeBalanceSlot(_ context.Context, token common.Address, _ ProbeStrategyExtraParams) (*entity.ERC20BalanceSlot, error) {
	m := map[common.Address]common.Hash{
		common.HexToAddress(btcbAddr):  common.HexToHash("0x4f1749155d837e5f5ef076382254c01af904c6ddb97b100fef402248f448ea99"),
		common.HexToAddress(wetheAddr): common.HexToHash("0x4f1749155d837e5f5ef076382254c01af904c6ddb97b100fef402248f448ea99"),
	}
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
	slot, ok := m[token]
	if !ok {
		return nil, errors.New("not found")
	}
	return &entity.ERC20BalanceSlot{
		Token:       strings.ToLower(token.String()),
		Wallet:      strings.ToLower(common.Address{}.String()),
		Found:       true,
		BalanceSlot: slot.String(),
	}, nil
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
	c := NewCache(repo, NewTestMultipleStrategy(&testProbe{}), nil)
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
	c := NewCache(repo, NewTestMultipleStrategy(&testProbe{}), nil)

	bl, err := c.Get(context.Background(), common.HexToAddress(btcbAddr), nil)
	require.NoError(t, err)
	require.NotEmptyf(t, bl.BalanceSlot, "must have balance slot")

	// set .probe to nil to test if the subsequent request uses cached balance slot
	origProbe := c.probe
	c.probe = nil
	bl, err = c.Get(context.Background(), common.HexToAddress(btcbAddr), nil)
	require.NoError(t, err)
	require.NotEmptyf(t, bl.BalanceSlot, "must have cached balance slot")
	c.probe = origProbe

	_, err = c.Get(context.Background(), common.HexToAddress(wetheAddr), nil)
	require.NoError(t, err)

	// must commit newly probed token to redis
	_, err = c.CommitToRedis(context.Background())
	require.NoError(t, err)

	bls, err := redisClient.HGetAll(context.Background(), utils.Join(prefix, erc20balanceslot.KeyERC20BalanceSlot)).Result()
	require.NoError(t, err)
	require.Truef(t, len(bls) == 2, "there must be 2 balance slots")
	for _, token := range []string{btcbAddr, wetheAddr} {
		rawBl, ok := bls[strings.ToLower(token)]
		require.Truef(t, ok, "must have token %s", token)
		bl := new(entity.ERC20BalanceSlot)
		err = json.Unmarshal([]byte(rawBl), bl)
		require.NoErrorf(t, err, "must unmarshal")
		require.Truef(t, bl.Found, "Found must be true")
		require.NotEmptyf(t, bl.BalanceSlot, "balance slot must available")
		require.EqualValuesf(t, []string{"test_probe"}, bl.StrategiesAttempted, "must record strategy attempted")
	}

	// must ignore failed strategy
	_entry, _ := c.cache.Load(common.HexToAddress(btcbAddr))
	entry := _entry.(*entity.ERC20BalanceSlot)
	entry.Found = false
	_, err = c.Get(context.Background(), common.HexToAddress(btcbAddr), nil)
	require.Error(t, err)
	fmt.Printf("%s\n", err)

	// subsequent commit should commit nothing
	numCommit, err := c.CommitToRedis(context.Background())
	require.NoError(t, err)
	require.Equalf(t, numCommit, 0, "there must nothing to commit")
}
