package pooltypes

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

type PoolCreatedDecoder struct {
	poolCreatedEventIds map[common.Hash]struct{}
	poolCreatedDecoder  func(log ethtypes.Log, exchange valueobject.Exchange) (*entity.Pool, error)
	poolType            string
	exchange            valueobject.Exchange
}

var decoderByPoolType = map[string]PoolCreatedDecoder{}
