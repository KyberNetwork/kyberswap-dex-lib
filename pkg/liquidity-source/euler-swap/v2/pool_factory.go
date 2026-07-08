package v2

import (
	"context"
	"errors"
	"math/big"
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
	return event == registryABI.Events["PoolRegistered"].ID
}

// DecodePoolCreated decodes the registry's PoolRegistered event. Unlike v1,
// the event already carries the pool's static params, so only the dynamic
// params and EVC address (not part of the event) need an RPC round trip.
func (f *PoolFactory) DecodePoolCreated(event types.Log) (*entity.Pool, error) {
	if len(event.Topics) != 4 || !strings.EqualFold(event.Address.Hex(), f.config.FactoryAddress) ||
		event.Topics[0] != registryABI.Events["PoolRegistered"].ID {
		return nil, errors.New("event is not supported")
	}

	var registered struct {
		Pool         common.Address     `abi:"pool"`
		SParams      StaticParamsFields `abi:"sParams"`
		ValidityBond *big.Int           `abi:"validityBond"`
	}
	if err := registryABI.UnpackIntoInterface(&registered, "PoolRegistered", event.Data); err != nil {
		return nil, err
	}

	asset0 := common.BytesToAddress(event.Topics[1].Bytes())
	asset1 := common.BytesToAddress(event.Topics[2].Bytes())

	var (
		evc           common.Address
		dynamicParams DynamicParamsRPC
	)

	req := f.ethrpcClient.NewRequest().SetContext(context.Background())
	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: registered.Pool.Hex(),
		Method: shared.PoolMethodEVC,
	}, []any{&evc})
	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: registered.Pool.Hex(),
		Method: shared.PoolMethodGetDynamicParams,
	}, []any{&dynamicParams})

	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}

	staticExtra := buildStaticExtra(registered.SParams, evc)
	staticExtraBytes, err := json.Marshal(&staticExtra)
	if err != nil {
		return nil, err
	}

	extra := Extra{
		Pause:         1, // unlocked
		DynamicParams: buildDynamicParams(dynamicParams.Data),
	}
	extraBytes, err := json.Marshal(&extra)
	if err != nil {
		return nil, err
	}

	tokens := []*entity.PoolToken{
		{Address: hexutil.Encode(asset0[:]), Swappable: true},
		{Address: hexutil.Encode(asset1[:]), Swappable: true},
	}

	return &entity.Pool{
		Address:     hexutil.Encode(registered.Pool[:]),
		Exchange:    f.config.DexID,
		Type:        DexType,
		Timestamp:   time.Now().Unix(),
		Reserves:    entity.PoolReserves{"0", "0"},
		Tokens:      tokens,
		Extra:       string(extraBytes),
		StaticExtra: string(staticExtraBytes),
		BlockNumber: event.BlockNumber,
	}, nil
}
