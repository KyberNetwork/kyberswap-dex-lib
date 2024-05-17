package route

import (
	"encoding/json"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func encodeRoute(route valueobject.SimpleRoute) (string, error) {
	bytes, err := json.Marshal(route)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func decodeRoute(data string) (*valueobject.SimpleRoute, error) {
	var route valueobject.SimpleRoute
	if err := json.Unmarshal([]byte(data), &route); err != nil {
		return nil, err
	}

	return &route, nil
}
