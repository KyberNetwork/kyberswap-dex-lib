package token

import (
	"context"
	"encoding/json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

func EncodeToken[T entity.SimplifiedToken | entity.Token](token T) (string, error) {
	bytes, err := json.Marshal(token)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func DecodeToken[T IToken](ctx context.Context, data string, addr string) (*T, error) {
	return decodeToken[T](ctx, data, addr)
}
