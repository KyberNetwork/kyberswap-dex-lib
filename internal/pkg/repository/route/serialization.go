package route

import (
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func encodeRoute(route valueobject.SimpleRouteWithExtraData) (string, error) {
	bytes, err := json.Marshal(route)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func decodeRoute(data string) (*valueobject.SimpleRouteWithExtraData, error) {
	var route valueobject.SimpleRouteWithExtraData
	if err := json.Unmarshal([]byte(data), &route); err != nil {
		return nil, err
	}

	return &route, nil
}
