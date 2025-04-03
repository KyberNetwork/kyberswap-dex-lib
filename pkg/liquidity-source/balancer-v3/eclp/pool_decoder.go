package eclp

import (
	"context"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/shared"
	pooldecoder "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/decode"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
)

type PoolDecoder struct {
	config *shared.Config
}

var _ = pooldecoder.RegisterFactoryC(DexType, NewPoolDecoder)

func NewPoolDecoder(config *shared.Config) *PoolDecoder {
	return &PoolDecoder{
		config: config,
	}
}

func (p *PoolDecoder) Decode(ctx context.Context, logs []types.Log) ([]string, error) {
	var poolAddresses []string
	for _, log := range logs {
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
			poolAddresses = append(poolAddresses, hexutil.Encode(log.Topics[1][:]))
		}
	}

	return poolAddresses, nil
}

func (p *PoolDecoder) GetKey(ctx context.Context) (poolAddress string, err error) {
	if p.config.Vault == "" {
		return "", errors.New("vault address is not set")
	}
	return strings.ToLower(p.config.Vault), nil
}
