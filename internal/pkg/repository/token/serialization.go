package token

import (
	"context"

	"github.com/goccy/go-json"
	"github.com/rs/zerolog/log"
)

// Do not support encode for SimplifiedToken because it is not backward compatible with json encoding in Redis token set.
func decodeToken[T IToken](ctx context.Context, data string, addr string) (T, error) {
	var token T
	if err := json.Unmarshal([]byte(data), &token); err != nil {
		return token, err
	}

	if tokenAddr := token.GetAddress(); tokenAddr != addr {
		log.Ctx(ctx).Error().Str("key", addr).Str("tokenAddr", tokenAddr).Msg("token address differs from hash key")
	}

	return token, nil
}
