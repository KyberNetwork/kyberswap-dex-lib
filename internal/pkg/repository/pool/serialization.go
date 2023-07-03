package pool

import (
	"encoding/json"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/pkg/mempool"
)

func encodePool(p entity.Pool) (string, error) {
	bytes, err := json.Marshal(p)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func decodePool(address string, data string) (*entity.Pool, error) {
	pool := mempool.EntityPool.Get().(*entity.Pool)
	// clear old data when get exist from mempool
	pool.Clear()
	if err := json.Unmarshal([]byte(data), pool); err != nil {
		return nil, err
	}

	pool.Address = address

	return pool, nil
}
