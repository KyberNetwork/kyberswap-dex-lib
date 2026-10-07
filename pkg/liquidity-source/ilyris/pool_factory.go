package ilyris

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
)

// PoolFactory decodes BinFactory.PoolCreated(address indexed tokenX, address indexed tokenY,
// uint24 indexed binStepBps, uint24 swapFeeBps, address policy, address marketGuard, address pool).
type PoolFactory struct {
	cfg *Config
}

var (
	_                          = poolfactory.RegisterFactoryC(DexType, NewPoolFactory)
	_ pool.IPoolFactoryDecoder = (*PoolFactory)(nil)

	// keccak256("PoolCreated(address,address,uint24,uint24,address,address,address)")
	poolCreatedTopic = common.HexToHash("0x953e8d471484a557f46b1052df28d856c12c8901507ad3410e3d2756e053e4a3")
)

func NewPoolFactory(cfg *Config) *PoolFactory {
	return &PoolFactory{cfg: cfg}
}

func (f *PoolFactory) IsEventSupported(hash common.Hash) bool {
	return hash == poolCreatedTopic
}

func (f *PoolFactory) DecodePoolCreated(ev types.Log) (*entity.Pool, error) {
	// The pool is the 4th non-indexed word; the emitter is the factory.
	if len(ev.Topics) < 4 || ev.Topics[0] != poolCreatedTopic || len(ev.Data) < 128 {
		return nil, ErrMalformedExtra
	}
	poolAddr := common.BytesToAddress(ev.Data[96:128])
	if poolAddr == (common.Address{}) {
		return nil, ErrMalformedExtra
	}
	tokenX, tokenY := common.BytesToAddress(ev.Topics[1][:]), common.BytesToAddress(ev.Topics[2][:])
	return &entity.Pool{
		Address:  hexutil.Encode(poolAddr[:]),
		Exchange: f.cfg.DexID,
		Type:     DexType,
		Reserves: entity.PoolReserves{"0", "0"},
		Tokens: []*entity.PoolToken{
			{Address: hexutil.Encode(tokenX[:]), Swappable: true},
			{Address: hexutil.Encode(tokenY[:]), Swappable: true},
		},
		BlockNumber: ev.BlockNumber,
	}, nil
}
