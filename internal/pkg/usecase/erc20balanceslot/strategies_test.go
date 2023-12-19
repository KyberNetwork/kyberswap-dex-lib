package erc20balanceslot

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	repo "github.com/KyberNetwork/router-service/internal/pkg/repository/erc20balanceslot"
	"github.com/KyberNetwork/router-service/pkg/logger"
	"github.com/KyberNetwork/router-service/pkg/redis"
)

const (
	jsonRPCURL = "http://localhost:8545"
)

func TestWholeSlotWithFStrategy(t *testing.T) {
	t.Skip()

	logger.InitLogger(logger.Configuration{
		EnableConsole: true,
		ConsoleLevel:  "debug",
	}, logger.LoggerBackendZap)

	rpcClient, err := rpc.Dial(jsonRPCURL)
	require.NoError(t, err)

	wallet := common.HexToAddress("0x19767032471665DF0FD7f6160381a103eCe6261A")
	p := NewWholeSlotWithFStrategy(rpcClient, wallet)

	token := common.HexToAddress("0x098754d293dd4375de7dC6566275aFb138021D00")
	bl, err := p.ProbeBalanceSlot(context.TODO(), token, nil)
	require.NoError(t, err)
	require.True(t, bl.Found)
	spew.Dump(bl)
}

func TestDoubleFromSourceStrategy(t *testing.T) {
	t.Skip()

	logger.InitLogger(logger.Configuration{
		EnableConsole: true,
		ConsoleLevel:  "debug",
	}, logger.LoggerBackendZap)

	rpcClient, err := rpc.Dial(jsonRPCURL)
	require.NoError(t, err)

	p := NewDoubleFromSourceStrategy(rpcClient)

	token := common.HexToAddress("0x5A98FcBEA516Cf06857215779Fd812CA3beF1B32")
	cloneSource := common.HexToAddress("0xa3f558aebAecAf0e11cA4b2199cC5Ed341edfd74")
	bl, err := p.ProbeBalanceSlot(context.TODO(), token, &DoubleFromSourceStrategyExtraParams{
		Source: cloneSource,
	})
	require.NoError(t, err)
	require.True(t, bl.Found)
	spew.Dump(bl)
}

func TestHoldersListStrategy(t *testing.T) {
	logger.InitLogger(logger.Configuration{
		EnableConsole: true,
		ConsoleLevel:  "debug",
	}, logger.LoggerBackendZap)

	miniRedis := miniredis.NewMiniRedis()
	err := miniRedis.Start()
	defer miniRedis.Close()
	require.NoError(t, err)

	redisClient, err := redis.New(&redis.Config{
		Addresses: []string{miniRedis.Addr()},
	})
	require.NoError(t, err)

	holdersListRepo := repo.NewHoldersListRedisRepositoryWithCache(redisClient, 60)
	watchlistRepo := repo.NewWatchlistRedisRepository(redisClient)

	p := NewHoldersListStrategy(randomizeAddress(), holdersListRepo, watchlistRepo)

	token := common.HexToAddress("0x974c5f5e3219a90bf2eb48188fa63fee9241a1fc")
	_, err = p.ProbeBalanceSlot(context.TODO(), token, nil)
	require.Errorf(t, err, "must error")

	holdersList := &entity.ERC20HoldersList{
		Token:   strings.ToLower(token.String()),
		Holders: []string{"aaa", "bbb"},
	}
	holdersListEncoded, _ := json.Marshal(holdersList)
	redisClient.Client.HSet(context.TODO(), redisClient.FormatKey(repo.KeyHoldersList), strings.ToLower(token.String()), holdersListEncoded).Val()

	bl, err := p.ProbeBalanceSlot(context.TODO(), token, nil)
	require.NoError(t, err)
	require.Equal(t, holdersList.Holders, bl.Holders)
}

func TestMultipleStrategy(t *testing.T) {
	t.Skip()

	logger.InitLogger(logger.Configuration{
		EnableConsole: true,
		ConsoleLevel:  "debug",
	}, logger.LoggerBackendZap)

	rpcClient, err := rpc.Dial(jsonRPCURL)
	require.NoError(t, err)

	p := NewMultipleStrategy(rpcClient, randomizeAddress())

	t.Run("must success with WholeSlotWithFStrategy", func(t *testing.T) {
		token := common.HexToAddress("0x098754d293dd4375de7dC6566275aFb138021D00")
		bl, err := p.ProbeBalanceSlot(context.TODO(), token, nil, &MultipleStrategyExtraParams{})
		require.NoError(t, err)
		require.True(t, bl.Found)
		require.NotEmpty(t, bl.StrategiesAttempted)
		require.Equal(t, "whole_slot_with_f", bl.StrategiesAttempted[len(bl.StrategiesAttempted)-1])
		spew.Dump(bl)
	})

	t.Run("must success with DoubleFromSourceStrategy", func(t *testing.T) {
		token := common.HexToAddress("0x5A98FcBEA516Cf06857215779Fd812CA3beF1B32")
		cloneSource := common.HexToAddress("0xa3f558aebAecAf0e11cA4b2199cC5Ed341edfd74")
		bl, err := p.ProbeBalanceSlot(context.TODO(), token, nil, &MultipleStrategyExtraParams{
			DoubleFromSource: &DoubleFromSourceStrategyExtraParams{Source: cloneSource},
		})
		require.NoError(t, err)
		require.True(t, bl.Found)
		require.NotEmpty(t, bl.StrategiesAttempted)
		require.Equal(t, fmt.Sprintf("double_from_source,source=%s", strings.ToLower(cloneSource.String())), bl.StrategiesAttempted[len(bl.StrategiesAttempted)-1])
		spew.Dump(bl)
	})
}
