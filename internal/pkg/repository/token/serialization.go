package token

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/goccy/go-json"
)

func encodeToken(token entity.Token) (string, error) {
	bytes, err := json.Marshal(token)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func decodeToken(address string, data string) (*entity.Token, error) {
	var token entity.Token
	if err := json.Unmarshal([]byte(data), &token); err != nil {
		return nil, err
	}

	token.Address = address

	return &token, nil
}
