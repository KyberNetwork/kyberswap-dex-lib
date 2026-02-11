package vault

import (
	"context"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
)

var _ = poolfactory.RegisterFactoryC(Type, NewPoolFactory)

type PoolFactory struct {
	cfg          *Config
	vaultAddress string
}

func NewPoolFactory(config *Config) *PoolFactory {
	return &PoolFactory{
		cfg:          config,
		vaultAddress: strings.ToLower(config.Vault),
	}
}

func (f *PoolFactory) DecodePoolCreated(event types.Log) (*entity.Pool, error) {
	return nil, nil
}

func (f *PoolFactory) IsEventSupported(event common.Hash) bool {
	return true
}

func (f *PoolFactory) DecodePoolAddress(ctx context.Context, log types.Log) ([]string, error) {
	if log.Address != common.HexToAddress(f.vaultAddress) {
		return nil, nil
	}
	switch log.Topics[0] {
	case vaultABI.Events["Swap"].ID,
		vaultABI.Events["AggregateSwapFeePercentageChanged"].ID,
		vaultABI.Events["AggregateYieldFeePercentageChanged"].ID,
		vaultABI.Events["Approval"].ID,
		vaultABI.Events["LiquidityAdded"].ID,
		vaultABI.Events["LiquidityRemoved"].ID,
		vaultABI.Events["PoolInitialized"].ID,
		vaultABI.Events["PoolPausedStateChanged"].ID,
		vaultABI.Events["PoolRecoveryModeStateChanged"].ID,
		vaultABI.Events["PoolRegistered"].ID,
		vaultABI.Events["SwapFeePercentageChanged"].ID,
		vaultABI.Events["Transfer"].ID,
		vaultABI.Events["VaultAuxiliary"].ID: // these events have the pool address in topic1
		if len(log.Topics) < 2 {
			return nil, nil
		}
		p := hexutil.Encode(log.Topics[1][common.HashLength-common.AddressLength:])
		return []string{p}, nil
	}
	return nil, nil
}
