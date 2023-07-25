package price

import (
	"encoding/json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

func encodePrice(price entity.Price) (string, error) {
	bytes, err := json.Marshal(price)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func decodePrice(address string, data string) (*entity.Price, error) {
	var price entity.Price
	if err := json.Unmarshal([]byte(data), &price); err != nil {
		return nil, err
	}

	price.Address = address

	return &price, nil
}
