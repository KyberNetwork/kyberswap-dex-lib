package ilyris

import (
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

var _ pool.IPoolFactoryDecoder = (*PoolFactory)(nil)

// poolCreatedTopic is keccak of
//
//	PoolCreated(address,address,uint24,uint24,address,address,address)
//
// Hard-coded rather than computed at init so this needs no ABI parse to answer
// IsEventSupported, which their dispatcher calls for every log it sees.
//
// VERIFIED against the live factory with `cast keccak`, not derived by eye. An earlier
// integration in this repo shipped two topic constants swapped, inferred from how often each
// appeared, and the code decoded happily -- wrong values, no error.
const poolCreatedTopic = "0x953e8d471484a557f46b1052df28d856c12c8901507ad3410e3d2756e053e4a3"

// PoolFactory turns a PoolCreated log into a pool their lister can store.
type PoolFactory struct {
	exchange string
}

func NewPoolFactory(exchange string) *PoolFactory {
	if exchange == "" {
		exchange = string(DexType)
	}
	return &PoolFactory{exchange: exchange}
}

func (f *PoolFactory) IsEventSupported(hash common.Hash) bool {
	return strings.EqualFold(hash.Hex(), poolCreatedTopic)
}

// DecodePoolCreated reads a BinFactory.PoolCreated log.
//
//	event PoolCreated(
//	    address indexed tokenX, address indexed tokenY, uint24 indexed binStepBps,
//	    uint24 swapFeeBps, address policy, address marketGuard, address pool)
//
// The log emitter is the factory. The pool address is the fourth non-indexed
// word (data[96:128]), matching abi/BinFactory.json. Keying the entity as the
// factory would collapse every pool into one row.
func (f *PoolFactory) DecodePoolCreated(ev types.Log) (*entity.Pool, error) {
	if len(ev.Topics) < 4 || !f.IsEventSupported(ev.Topics[0]) {
		return nil, ErrMalformedExtra
	}
	poolAddr, err := poolCreatedAddress(ev.Data)
	if err != nil {
		return nil, err
	}
	tokenX := strings.ToLower(common.BytesToAddress(ev.Topics[1].Bytes()).Hex())
	tokenY := strings.ToLower(common.BytesToAddress(ev.Topics[2].Bytes()).Hex())

	return &entity.Pool{
		Address:  strings.ToLower(poolAddr.Hex()),
		Exchange: f.exchange,
		Type:     string(DexType),
		Tokens: []*entity.PoolToken{
			{Address: tokenX, Swappable: true},
			{Address: tokenY, Swappable: true},
		},
		BlockNumber: ev.BlockNumber,
	}, nil
}

// poolCreatedAddress is data word 3 of PoolCreated: swapFeeBps, policy,
// marketGuard, then pool. Each word is 32 bytes, so the pool starts at byte 96.
func poolCreatedAddress(data []byte) (common.Address, error) {
	if len(data) < 128 {
		return common.Address{}, ErrMalformedExtra
	}
	addr := common.BytesToAddress(data[96:128])
	if addr == (common.Address{}) {
		return common.Address{}, ErrMalformedExtra
	}
	return addr, nil
}
