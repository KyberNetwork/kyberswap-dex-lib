package dexv2

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/fluid/dex-v2/abis"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var _ = poolfactory.RegisterFactoryC(DexType, NewPoolFactory)

type PoolFactory struct {
	config              *Config
	poolCreatedEventIds map[common.Hash]struct{}
}

func NewPoolFactory(config *Config) *PoolFactory {
	return &PoolFactory{
		config: config,
		poolCreatedEventIds: map[common.Hash]struct{}{
			abis.DexV2ABI.Events["LogInitialize"].ID: {},
		},
	}
}

func (f *PoolFactory) DecodePoolCreated(event ethtypes.Log) (*entity.Pool, error) {
	p, err := abis.DexV2PoolFilterer.ParseLogInitialize(event)
	if err != nil {
		return nil, err
	}

	return f.newPool(p, event.BlockNumber)
}

func (f *PoolFactory) IsEventSupported(event common.Hash) bool {
	_, ok := f.poolCreatedEventIds[event]
	return ok
}

func (f *PoolFactory) newPool(p *abis.FluidDexV2LogInitialize, blockNumber uint64) (*entity.Pool, error) {
	poolAddress := encodeFluidDexV2PoolAddress(hexutil.Encode(p.DexId[:]), uint32(p.DexType.Uint64()))

	token0 := entity.PoolToken{
		Address:   hexutil.Encode(p.DexKey.Token0[:]),
		Swappable: true,
	}
	token1 := entity.PoolToken{
		Address:   hexutil.Encode(p.DexKey.Token1[:]),
		Swappable: true,
	}

	tokens := []*entity.PoolToken{&token0, &token1}
	isNative := [2]bool{false, false}
	for i, token := range tokens {
		if valueobject.IsNative(token.Address) {
			tokens[i].Address = valueobject.WrapNativeLower(token.Address, valueobject.ChainID(f.config.ChainID))
			isNative[i] = true
		}
	}

	staticExtra := StaticExtra{
		Dex:         f.config.Dex,
		DexType:     uint32(p.DexType.Uint64()),
		Fee:         uint32(p.DexKey.Fee.Uint64()),
		TickSpacing: uint32(p.DexKey.TickSpacing.Uint64()),
		IsNative:    isNative,
	}
	if p.DexKey.Controller != valueobject.AddrZero {
		staticExtra.Controller = p.DexKey.Controller.String()
	}
	staticExtraBytes, err := json.Marshal(staticExtra)
	if err != nil {
		return nil, err
	}

	return &entity.Pool{
		Address:     poolAddress,
		Exchange:    f.config.DexID,
		Type:        DexType,
		Reserves:    []string{"0", "0"},
		Tokens:      tokens,
		StaticExtra: string(staticExtraBytes),
		Extra:       "{}",
		BlockNumber: blockNumber,
		Timestamp:   time.Now().Unix(),
	}, nil
}

func (f *PoolFactory) DecodePoolAddress(log ethtypes.Log) (string, error) {
	if len(log.Topics) == 0 || eth.IsZeroAddress(log.Address) {
		return "", nil
	}

	dexId, dexType, err := func() ([32]byte, *big.Int, error) {
		switch log.Topics[0] {
		case abis.DexV2ABI.Events["LogInitialize"].ID:
			p, err := abis.DexV2PoolFilterer.ParseLogInitialize(log)
			if err != nil {
				return [32]byte{}, nil, err
			}

			return p.DexId, p.DexType, nil

		case abis.DexV2ABI.Events["LogDeposit"].ID:
			p, err := abis.DexV2PoolFilterer.ParseLogDeposit(log)
			if err != nil {
				return [32]byte{}, nil, err
			}

			return p.DexId, p.DexType, nil

		case abis.DexV2ABI.Events["LogWithdraw"].ID:
			p, err := abis.DexV2PoolFilterer.ParseLogWithdraw(log)
			if err != nil {
				return [32]byte{}, nil, err
			}

			return p.DexId, p.DexType, nil

		case abis.DexV2ABI.Events["LogBorrow"].ID:
			p, err := abis.DexV2PoolFilterer.ParseLogBorrow(log)
			if err != nil {
				return [32]byte{}, nil, err
			}

			return p.DexId, p.DexType, nil

		case abis.DexV2ABI.Events["LogPayback"].ID:
			p, err := abis.DexV2PoolFilterer.ParseLogPayback(log)
			if err != nil {
				return [32]byte{}, nil, err
			}

			return p.DexId, p.DexType, nil

		default:
			return [32]byte{}, nil, nil
		}
	}()

	if err != nil || dexType == nil {
		return "", err
	}

	return encodeFluidDexV2PoolAddress(hexutil.Encode(dexId[:]), uint32(dexType.Uint64())), nil
}
