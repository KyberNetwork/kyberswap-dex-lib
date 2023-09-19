package encode

import "github.com/KyberNetwork/router-service/internal/pkg/usecase/types"

type IEncoder interface {
	Encode(data types.EncodingData) (string, error)
	GetExecutorAddress() string
	GetRouterAddress() string
	GetKyberLOAddress() string
}
