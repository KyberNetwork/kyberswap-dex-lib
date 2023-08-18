package erc20balanceslot

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/pkg/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/redis/go-redis/v9"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
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

func (r *RedisRepository) GetPrefix() string {
	return r.prefix
}

func (r *RedisRepository) Get(ctx context.Context, token common.Address) (*entity.ERC20BalanceSlot, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[erc20balanceslot] redisRepository.Get")
	defer span.Finish()

	rawResult := r.redisClient.HGet(ctx, r.redisKey, strings.ToLower(token.String())).Val()
	if rawResult == "" {
		return nil, fmt.Errorf("balance slot for token %s not found", token)
	}

	result := new(entity.ERC20BalanceSlot)
	if err := json.Unmarshal([]byte(rawResult), result); err != nil {
		logger.WithFields(logger.Fields{"token": token}).Warn("could not unmarshal entity.ERC20BalanceSlot")
		return nil, err
	}

	return result, nil
}

func (r *RedisRepository) GetAll(ctx context.Context) (map[common.Address]*entity.ERC20BalanceSlot, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[erc20balanceslot] redisRepository.GetAll")
	defer span.Finish()

	rawResult := r.redisClient.HGetAll(ctx, r.redisKey).Val()
	result := make(map[common.Address]*entity.ERC20BalanceSlot)
	for token, rawValue := range rawResult {
		token = strings.ToLower(token)
		balanceSlot := new(entity.ERC20BalanceSlot)
		if err := json.Unmarshal([]byte(rawValue), balanceSlot); err != nil {
			logger.WithFields(logger.Fields{"token": token}).Warn("could not unmarshal entity.ERC20BalanceSlot")
			continue
		}
		result[common.HexToAddress(token)] = balanceSlot
	}

	return result, nil
}

func (r *RedisRepository) PutMany(ctx context.Context, balanceSlots []*entity.ERC20BalanceSlot) error {
	span, ctx := tracer.StartSpanFromContext(ctx, "[erc20balanceslot] redisRepository.Put")
	defer span.Finish()

	if len(balanceSlots) == 0 {
		return nil
	}

	pipe := r.redisClient.Pipeline()
	for _, bl := range balanceSlots {
		encoded, err := json.Marshal(bl)
		if err != nil {
			logger.WithFields(logger.Fields{"entity": bl}).Warn("could not marshal entity.ERC20BalanceSlot")
			return err
		}
		pipe.HSet(ctx, r.redisKey, strings.ToLower(bl.Token), string(encoded))
	}
	_, err := pipe.Exec(ctx)

	if err != nil {
		logger.WithFields(logger.Fields{"err": err}).Warn("could not exect multiple HMSET")
		return err
	}

	return nil
}
