package pool

import (
	"encoding/json"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
)

func encodePool(p entity.Pool) (string, error) {
	bytes, err := json.Marshal(p)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func decodePool(address string, data string) (*entity.Pool, error) {
	var pool entity.Pool
	if err := json.Unmarshal([]byte(data), &pool); err != nil {
		return nil, err
	}

	pool.Address = address

	return &pool, nil
}
