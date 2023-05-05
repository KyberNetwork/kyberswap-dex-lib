package pool

import (
	"encoding/json"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
)

func decodePool(address string, data string) (*entity.Pool, error) {
	var pool entity.Pool
	if err := json.Unmarshal([]byte(data), &pool); err != nil {
		return nil, err
	}

	pool.Address = address

	return &pool, nil
}
