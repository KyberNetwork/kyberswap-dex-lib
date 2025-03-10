package route

import (
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/dgraph-io/ristretto"
)

func EncodeRoute(route valueobject.SimpleRouteWithExtraData) (string, error) {
	return encodeRoute(route)
}

func DecodeRoute(data string) (*valueobject.SimpleRouteWithExtraData, error) {
	return decodeRoute(data)
}

func (r ristrettoRepository) Cache() *ristretto.Cache {
	return r.cache
}
