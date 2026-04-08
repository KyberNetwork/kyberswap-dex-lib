package stablestable

import (
	"context"

	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var (
	_ = poolfactory.RegisterStaticFactory(newPoolFactory(), HookAddresses...)
)

type PoolFactory struct {
	pool.FactoryDecoder
}

func newPoolFactory() *PoolFactory {
	return &PoolFactory{}
}

func (f *PoolFactory) DecodePoolAddressesFromFactoryLog(_ context.Context, log ethtypes.Log) ([]string, error) {
	if len(log.Topics) == 0 || valueobject.IsZeroAddress(log.Address) {
		return nil, nil
	}

	switch log.Topics[0] {
	case stableStableHookABI.Events["FeeConfigUpdated"].ID:
		return []string{hexutil.Encode(log.Topics[1][:])}, nil
	}

	return nil, nil
}
