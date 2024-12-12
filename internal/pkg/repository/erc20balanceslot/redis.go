package erc20balanceslot

import (
	"context"
	"fmt"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/redis/go-redis/v9"

	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

const (
	HashKeyReadChunkSize = 10_000
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

func (r *RedisRepository) Count(ctx context.Context) (int, error) {
	count, err := r.redisClient.HLen(ctx, r.redisKey).Result()
	return int(count), err
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

func (r *RedisRepository) GetMany(ctx context.Context, tokens []common.Address) (map[common.Address]*types.ERC20BalanceSlot, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[erc20balanceslot] redisRepository.GetMany")
	defer span.End()

	numChunks := len(tokens) / HashKeyReadChunkSize
	if len(tokens)%HashKeyReadChunkSize != 0 {
		numChunks++
	}
	chunkedKeys := make([][]string, 0, numChunks)
	for c := 0; c < numChunks; c++ {
		start := c * HashKeyReadChunkSize
		end := start + HashKeyReadChunkSize
		if end > len(tokens) {
			end = len(tokens)
		}
		keys := make([]string, 0, end-start)
		for i := start; i < end; i++ {
			keys = append(keys, strings.ToLower(tokens[i].String()))
		}
		chunkedKeys = append(chunkedKeys, keys)
	}

	cmds, _ := r.redisClient.Pipelined(ctx, func(p redis.Pipeliner) error {
		for c := 0; c < numChunks; c++ {
			p.HMGet(ctx, r.redisKey, chunkedKeys[c]...)
		}
		return nil
	})
	rawResults := make([]interface{}, len(tokens))
	for _, cmd := range cmds {
		sliceCmd, ok := cmd.(*redis.SliceCmd)
		if !ok {
			return nil, fmt.Errorf("expected *redis.SliceCmd")
		}
		chunkResults, err := sliceCmd.Result()
		if err != nil {
			return nil, fmt.Errorf("pipelined HMGET returns error: %w", err)
		}
		rawResults = append(rawResults, chunkResults...)
	}

	results := make(map[common.Address]*types.ERC20BalanceSlot, len(rawResults))
	for _, rawResult := range rawResults {
		if rawResult == nil {
			continue
		}
		resultStr, ok := rawResult.(string)
		if !ok {
			logger.Warn(ctx, "result should be string")
			continue
		}
		result := new(types.ERC20BalanceSlot)
		if err := json.Unmarshal([]byte(resultStr), result); err != nil {
			logger.Warn(ctx, "[erc20balanceslot] Get could not unmarshal entity.ERC20BalanceSlot")
			continue
		}
		results[common.HexToAddress(result.Token)] = result
	}

	return results, nil
}

func (r *RedisRepository) GetAll(ctx context.Context) (map[common.Address]*types.ERC20BalanceSlot, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[erc20balanceslot] redisRepository.GetAll")
	defer span.End()

	result := make(map[common.Address]*types.ERC20BalanceSlot)
	cursor := uint64(0)
	for {
		keyValues, newCursor, err := r.redisClient.HScan(ctx, r.redisKey, cursor, "", HashKeyReadChunkSize).Result()
		if err != nil {
			return nil, err
		}

		for i := 0; i < len(keyValues); i += 2 {
			token := strings.ToLower(keyValues[i])
			balanceSlot := new(types.ERC20BalanceSlot)
			if err := json.Unmarshal([]byte(keyValues[i+1]), balanceSlot); err != nil {
				logger.WithFields(ctx, logger.Fields{"token": token}).Warn("could not unmarshal entity.ERC20BalanceSlot")
				continue
			}
			result[common.HexToAddress(token)] = balanceSlot
		}

		cursor = newCursor
		if cursor == 0 {
			break
		}
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
