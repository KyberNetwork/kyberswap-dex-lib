package vault

import (
	"context"
	"strings"

	pooldecoder "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/decode"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
)

type Config struct {
	Vault string `json:"vault,omitempty"`
}

type EventParser struct {
	config *Config
}

var _ = pooldecoder.RegisterFactoryC(Type, NewEventParser)

func NewEventParser(config *Config) *EventParser {
	return &EventParser{
		config: config,
	}
}

func (p *EventParser) Decode(ctx context.Context, logs []types.Log) (map[string][]types.Log, error) {
	vaultAddress, err := p.GetKey(ctx)
	if err != nil {
		return nil, err
	}
	addressLogs := make(map[string][]types.Log)
	for _, log := range logs {
		if log.Address != common.HexToAddress(vaultAddress) {
			continue
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
				break
			}
			p := strings.ToLower(common.HexToAddress(hexutil.Encode(log.Topics[1][:])).Hex())
			addressLogs[p] = append(addressLogs[p], log)
		}
	}

	return addressLogs, nil
}

func (p *EventParser) GetKey(ctx context.Context) (poolAddress string, err error) {
	if p.config.Vault == "" {
		return "", errors.New("vault address is not set")
	}
	return strings.ToLower(p.config.Vault), nil
}
