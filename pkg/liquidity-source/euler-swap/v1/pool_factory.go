package v1

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/euler-swap/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
)

var _ = poolfactory.RegisterFactoryCE(DexType, NewPoolFactory)

type PoolFactory struct {
	config       *shared.Config
	ethrpcClient *ethrpc.Client
}

func NewPoolFactory(config *shared.Config, ethrpcClient *ethrpc.Client) *PoolFactory {
	return &PoolFactory{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (f *PoolFactory) IsEventSupported(event common.Hash) bool {
	return event == factoryABI.Events["PoolDeployed"].ID
}

// DecodePoolCreated decodes the factory's PoolDeployed event. The event only
// carries the tokens and pool address, so the pool's immutable params (vaults,
// EulerAccount, EVC, ...) are still fetched via RPC, same as PoolsListUpdater.
func (f *PoolFactory) DecodePoolCreated(event types.Log) (*entity.Pool, error) {
	if len(event.Topics) != 4 || !strings.EqualFold(event.Address.Hex(), f.config.FactoryAddress) ||
		event.Topics[0] != factoryABI.Events["PoolDeployed"].ID {
		return nil, errors.New("event is not supported")
	}

	var deployed struct {
		Pool common.Address `abi:"pool"`
	}
	if err := factoryABI.UnpackIntoInterface(&deployed, "PoolDeployed", event.Data); err != nil {
		return nil, err
	}

	asset0 := common.BytesToAddress(event.Topics[1].Bytes())
	asset1 := common.BytesToAddress(event.Topics[2].Bytes())

	staticExtra, err := getPoolStaticData(context.Background(), f.ethrpcClient, deployed.Pool.Hex())
	if err != nil {
		return nil, err
	}

	extraBytes, err := json.Marshal(&staticExtra)
	if err != nil {
		return nil, err
	}

	token0 := &entity.PoolToken{
		Address:   hexutil.Encode(asset0[:]),
		Swappable: true,
	}
	token1 := &entity.PoolToken{
		Address:   hexutil.Encode(asset1[:]),
		Swappable: true,
	}

	return &entity.Pool{
		Address:     hexutil.Encode(deployed.Pool[:]),
		Exchange:    f.config.DexID,
		Type:        DexType,
		Timestamp:   time.Now().Unix(),
		Reserves:    []string{"0", "0"},
		Tokens:      []*entity.PoolToken{token0, token1},
		StaticExtra: string(extraBytes),
		BlockNumber: event.BlockNumber,
	}, nil
}
