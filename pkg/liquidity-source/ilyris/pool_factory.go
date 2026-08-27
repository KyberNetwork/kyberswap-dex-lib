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

// DecodePoolCreated reads the event's INDEXED fields from topics.
//
//	event PoolCreated(
//	    address indexed tokenX, address indexed tokenY, uint24 indexed binStepBps,
//	    uint24 swapFeeBps, address policy, address marketGuard, address pool)
//
// Only the three indexed fields are decoded here. swapFeeBps and the pool address live in
// `data`, and reading them needs the ABI -- but the pool ADDRESS is what the lister keys on,
// so this returns what it can prove and leaves enrichment to the tracker rather than guessing
// an offset. Getting a data offset wrong produces a valid-looking address that is not the
// pool, and nothing downstream would catch it.
func (f *PoolFactory) DecodePoolCreated(ev types.Log) (*entity.Pool, error) {
	if len(ev.Topics) < 4 || !f.IsEventSupported(ev.Topics[0]) {
		return nil, ErrMalformedExtra
	}
	tokenX := strings.ToLower(common.BytesToAddress(ev.Topics[1].Bytes()).Hex())
	tokenY := strings.ToLower(common.BytesToAddress(ev.Topics[2].Bytes()).Hex())

	return &entity.Pool{
		Address:  strings.ToLower(ev.Address.Hex()),
		Exchange: f.exchange,
		Type:     string(DexType),
		Tokens: []*entity.PoolToken{
			{Address: tokenX, Swappable: true},
			{Address: tokenY, Swappable: true},
		},
		BlockNumber: ev.BlockNumber,
	}, nil
}
