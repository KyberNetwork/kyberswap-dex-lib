package vault

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
)

type Config struct {
	Vault string `json:"vault,omitempty"`
}

type EventParser struct {
	config *Config
}

var _ = poolfactory.RegisterFactoryC(Type, NewPoolFactory)

func NewPoolFactory(config *Config) *EventParser {
	return &EventParser{
		config: config,
	}
}

func (p *EventParser) Decode(ctx context.Context, logs []types.Log) (map[string][]types.Log, error) {
	addressLogs := make(map[string][]types.Log)
	for _, log := range logs {
		addresses, _ := p.DecodePoolAddressesFromFactoryLog(ctx, log)
		for _, address := range addresses {
			addressLogs[address] = append(addressLogs[address], log)
		}
	}
	return addressLogs, nil
}

func (p *EventParser) DecodePoolAddressesFromFactoryLog(ctx context.Context, log types.Log) ([]string, error) {
	if log.Address != common.HexToAddress(p.config.Vault) {
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

func (p *EventParser) DecodePoolCreated(event types.Log) (*entity.Pool, error) {
	// TODO: Implement this (non tick-based pool creation)
	return nil, nil
}

func (p *EventParser) IsEventSupported(event common.Hash) bool {
	// TODO: Implement this (non tick-based pool creation)
	return true
}
