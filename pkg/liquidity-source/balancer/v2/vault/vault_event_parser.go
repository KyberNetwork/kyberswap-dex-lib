package vault

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v2/shared"
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

// Ported from the Balancer V3 vault event parser, adapted for V2.
// https://github.com/KyberNetwork/kyberswap-dex-lib/blob/0b94bbfeae66b43cb64a52308bf52520145ab79c/pkg/liquidity-source/balancer/v3/vault/vault_event_parser.go#L31-L67
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

// DecodePoolAddressesFromFactoryLog extracts the affected pool address from a V2 Vault log.
//
// Unlike the V3 Vault (which emits a left-padded 32-byte pool address in topic1 and is
// decoded via topic1[12:32]), the Balancer V2 Vault emits the pool's bytes32 poolId in
// topic1. The poolId is abi.encodePacked(pool address (20B), specialization (2B), nonce
// (10B)), so the pool address is the HIGH-ORDER 20 bytes of the poolId, i.e. topic1[0:20].
//
// Ported/mirrored from V3 DecodePoolAddressesFromFactoryLog:
// https://github.com/KyberNetwork/kyberswap-dex-lib/blob/0b94bbfeae66b43cb64a52308bf52520145ab79c/pkg/liquidity-source/balancer/v3/vault/vault_event_parser.go#L42-L67
func (p *EventParser) DecodePoolAddressesFromFactoryLog(_ context.Context, log types.Log) ([]string, error) {
	if log.Address != common.HexToAddress(p.config.Vault) {
		return nil, nil
	}
	switch log.Topics[0] {
	case shared.VaultABI.Events["Swap"].ID,
		shared.VaultABI.Events["PoolBalanceChanged"].ID,
		shared.VaultABI.Events["PoolBalanceManaged"].ID: // these events carry the poolId in topic1
		if len(log.Topics) < 2 {
			return nil, nil
		}
		// poolId = abi.encodePacked(pool address (20B), specialization (2B), nonce (10B));
		// the pool address is the high-order 20 bytes of the poolId.
		poolAddress := hexutil.Encode(log.Topics[1][:common.AddressLength])
		return []string{poolAddress}, nil
	}
	return nil, nil
}

func (ep *EventParser) DecodePoolCreated(event types.Log) (*entity.Pool, error) {
	// TODO: Implement this (non tick-based pool creation)
	return nil, errors.New("not implemented")
}

func (ep *EventParser) IsEventSupported(event common.Hash) bool {
	// TODO: Implement this (non tick-based pool creation)
	return false
}
