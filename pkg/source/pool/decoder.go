package pool

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type FactoryDecoder struct{}

func (f *FactoryDecoder) DecodePoolCreated(_ types.Log) (*entity.Pool, error) {
	return nil, nil
}

func (f *FactoryDecoder) IsEventSupported(_ common.Hash) bool {
	return false
}
