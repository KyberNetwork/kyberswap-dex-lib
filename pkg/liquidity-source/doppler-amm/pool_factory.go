package doppleramm

import (
	"context"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
)

// Topic0 of the AMM events whose topic1 is the market id. The AMM is unverified, so the hashes
// come from its live logs: a swap emits the first, liquidity add and remove emit the others.
var marketStateTopics = map[common.Hash]struct{}{
	common.HexToHash("0x07af4e23b8e693b486c551709747254c747c07d7f9b3930a72bb6b0c3ac5af83"): {}, // swap
	common.HexToHash("0xdb56780297c6bee5d6fcac88e28b683fc68cb5d85fea9205cd3ce956e8481e19"): {}, // liquidity add
	common.HexToHash("0x33c06bd624e4eb0d07c026198804af254b48b0a4d48f63a88e7a1e967ab7a983"): {}, // liquidity remove
}

var _ = poolfactory.RegisterFactoryC(DexType, NewPoolFactory)

// PoolFactory routes the AMM singleton's market events to the market's pool. Pools are
// discovered by the lister, so it creates none.
type PoolFactory struct {
	pool.FactoryDecoder
	amm common.Address
}

func NewPoolFactory(cfg *Config) *PoolFactory {
	amm := cfg.AMMAddress
	if d, ok := deployments[cfg.ChainID]; ok && amm == "" {
		amm = d.AMM
	}
	return &PoolFactory{amm: common.HexToAddress(amm)}
}

// DecodePoolAddressesFromFactoryLog maps a swap or liquidity log to its market id, the pool address.
func (f *PoolFactory) DecodePoolAddressesFromFactoryLog(_ context.Context, log types.Log) ([]string, error) {
	if log.Address != f.amm || len(log.Topics) < 2 {
		return nil, nil
	}
	if _, ok := marketStateTopics[log.Topics[0]]; !ok {
		return nil, nil
	}
	return []string{strings.ToLower(hexutil.Encode(log.Topics[1][:]))}, nil
}
