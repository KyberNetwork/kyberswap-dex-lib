package erc20balanceslot

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/redis/go-redis/v9"

	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type RedisRepository struct {
	redisClient redis.UniversalClient
	prefix      string
	redisKey    string
}

func NewRedisRepository(redisClient redis.UniversalClient, config RedisRepositoryConfig) *RedisRepository {
	return &RedisRepository{
		redisClient: redisClient,
		prefix:      config.Prefix,
		redisKey:    utils.Join(config.Prefix, KeyERC20BalanceSlot),
	}
}

func (r *RedisRepository) Get(ctx context.Context, token common.Address) (*types.ERC20BalanceSlot, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[erc20balanceslot] redisRepository.Get")
	defer span.End()

	rawResult := r.redisClient.HGet(ctx, r.redisKey, strings.ToLower(token.String())).Val()
	if rawResult == "" {
		return nil, fmt.Errorf("balance slot for token %s not found", token)
	}

	result := new(types.ERC20BalanceSlot)
	if err := json.Unmarshal([]byte(rawResult), result); err != nil {
		return nil, fmt.Errorf("[erc20balanceslot] Get could not unmarshal entity.ERC20BalanceSlot token %s", token)
	}

	return result, nil
}

func (r *RedisRepository) GetAll(ctx context.Context) (map[common.Address]*types.ERC20BalanceSlot, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[erc20balanceslot] redisRepository.GetAll")
	defer span.End()

	rawResult := r.redisClient.HGetAll(ctx, r.redisKey).Val()
	result := make(map[common.Address]*types.ERC20BalanceSlot)
	for token, rawValue := range rawResult {
		token = strings.ToLower(token)
		balanceSlot := new(types.ERC20BalanceSlot)
		if err := json.Unmarshal([]byte(rawValue), balanceSlot); err != nil {
			logger.WithFields(ctx, logger.Fields{"token": token}).Warn("could not unmarshal entity.ERC20BalanceSlot")
			continue
		}
		result[common.HexToAddress(token)] = balanceSlot
	}

	return result, nil
}

func (r *RedisRepository) Put(ctx context.Context, balanceSlot *types.ERC20BalanceSlot) error {
	encoded, err := json.Marshal(balanceSlot)
	if err != nil {
		return err
	}
	_, err = r.redisClient.HSet(ctx, r.redisKey, strings.ToLower(balanceSlot.Token), string(encoded)).Result()
	return err
}

func (r *RedisRepository) PutMany(ctx context.Context, balanceSlots []*types.ERC20BalanceSlot) error {
	span, ctx := tracer.StartSpanFromContext(ctx, "[erc20balanceslot] redisRepository.Put")
	defer span.End()

	if len(balanceSlots) == 0 {
		return nil
	}

	pipe := r.redisClient.Pipeline()
	for _, bl := range balanceSlots {
		encoded, err := json.Marshal(bl)
		if err != nil {
			return err
		}
		pipe.HSet(ctx, r.redisKey, strings.ToLower(bl.Token), string(encoded))
	}
	_, err := pipe.Exec(ctx)

	if err != nil {
		return err
	}

	return nil
}
