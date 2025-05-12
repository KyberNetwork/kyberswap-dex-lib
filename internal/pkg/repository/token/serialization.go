package token

import (
	"context"

	"github.com/KyberNetwork/router-service/pkg/logger"
	"github.com/goccy/go-json"
)

// Do not support endcode for SimplifiedToken because it is not backward compatible with json encoding in Redis token set.
func decodeToken[T IToken](ctx context.Context, data string, addr string) (*T, error) {
	var token T
	if err := json.Unmarshal([]byte(data), &token); err != nil {
		return nil, err
	}
	if (token).GetAddress() != addr {
		logger.Errorf(ctx, "token address differs from hash key %s token %v", addr, token)
	}

	return &token, nil
}
