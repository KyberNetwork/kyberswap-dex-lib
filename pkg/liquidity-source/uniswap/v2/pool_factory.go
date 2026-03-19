package uniswapv2

import (
	"errors"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2/abis"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
)

var _ = poolfactory.RegisterFactoryC(DexType, NewPoolFactory)

type PoolFactory struct {
	config *Config
}

func NewPoolFactory(config *Config) *PoolFactory {
	return &PoolFactory{
		config: config,
	}
}

func (f *PoolFactory) DecodePoolCreated(event types.Log) (*entity.Pool, error) {
	if len(event.Topics) == 0 || eth.IsZeroAddress(event.Address) || !strings.EqualFold(hexutil.Encode(event.Address[:]), f.config.FactoryAddress) {
		return nil, errors.New("event is not supported")
	}

	switch event.Topics[0] {
	case uniswapV2FactoryABI.Events["PairCreated"].ID:
		pool, err := uniswapV2FactoryFilterer.ParsePairCreated(event)
		if err != nil {
			return nil, err
		}
		return f.newPool(pool, event.BlockNumber)
	default:
		return nil, errors.New("event is not supported")
	}
}

func (f *PoolFactory) IsEventSupported(event common.Hash) bool {
	return event == uniswapV2FactoryABI.Events["PairCreated"].ID
}

func (f *PoolFactory) newPool(p *uniswapv2.UniswapV2FactoryPairCreated, blockNbr uint64) (*entity.Pool, error) {
	poolAddress := hexutil.Encode(p.Pair[:])

	token0 := entity.PoolToken{
		Address:   hexutil.Encode(p.Token0[:]),
		Swappable: true,
	}
	token1 := entity.PoolToken{
		Address:   hexutil.Encode(p.Token1[:]),
		Swappable: true,
	}
	reserves := entity.PoolReserves{
		"0", "0",
	}

	extra := Extra{
		Fee:          f.config.Fee,
		FeePrecision: f.config.FeePrecision,
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return nil, err
	}

	return &entity.Pool{
		Address:     poolAddress,
		Exchange:    f.config.DexID,
		Type:        DexType,
		Timestamp:   time.Now().Unix(),
		Reserves:    reserves,
		Tokens:      []*entity.PoolToken{&token0, &token1},
		Extra:       string(extraBytes),
		BlockNumber: blockNbr,
	}, nil
}
