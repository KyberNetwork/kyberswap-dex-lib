package token

import (
	"context"

	"github.com/goccy/go-json"
	"github.com/rs/zerolog/log"
)

// Do not support endcode for SimplifiedToken because it is not backward compatible with json encoding in Redis token set.
func decodeToken[T IToken](ctx context.Context, data string, addr string) (*T, error) {
	var token T
	if err := json.Unmarshal([]byte(data), &token); err != nil {
		return nil, err
	}
	if (token).GetAddress() != addr {
		log.Ctx(ctx).Error().Msgf("token address differs from hash key %s token %v", addr, token)
	}

	return &token, nil
}
