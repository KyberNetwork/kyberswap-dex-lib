package flywheelfun

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	config Config
	rpc    *ethrpc.Client
}
type listCursor struct {
	Offset uint64      `json:"offset"`
	Block  uint64      `json:"block"`
	Hash   common.Hash `json:"hash"`
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(c *Config, r *ethrpc.Client) *PoolsListUpdater {
	if c == nil {
		c = &Config{}
	}
	cfg := *c
	if cfg.NewPoolLimit <= 0 || cfg.NewPoolLimit > 100 {
		cfg.NewPoolLimit = 20
	}
	if cfg.DexID == "" {
		cfg.DexID = DexType
	}
	return &PoolsListUpdater{config: cfg, rpc: r}
}
func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadata []byte) ([]entity.Pool, []byte, error) {
	var cursor listCursor
	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &cursor); err != nil {
			return nil, metadata, err
		}
	}
	chain, err := u.rpc.GetETHClient().ChainID(ctx)
	if err != nil {
		return nil, metadata, err
	}
	if chain.Cmp(big.NewInt(4663)) != 0 {
		return nil, metadata, ErrState
	}
	if cursor.Block > 0 {
		h, e := u.rpc.GetETHClient().HeaderByNumber(ctx, new(big.Int).SetUint64(cursor.Block))
		if e != nil {
			return nil, metadata, e
		}
		if h.Hash() != cursor.Hash {
			cursor = listCursor{}
		} // Replay discovery idempotently after a reorg.
	}
	head, err := u.rpc.GetETHClient().HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, metadata, err
	}
	block := head.Number
	var count *big.Int
	_, err = u.rpc.NewRequest().SetContext(ctx).SetBlockNumber(block).AddCall(&ethrpc.Call{ABI: factoryABI, Target: Factory, Method: "allLaunchesCount"}, []any{&count}).Call()
	if err != nil {
		return nil, metadata, err
	}
	if count == nil || !count.IsUint64() {
		return nil, metadata, ErrState
	}
	if cursor.Offset > count.Uint64() {
		cursor.Offset = 0
	}
	size := min(uint64(u.config.NewPoolLimit), count.Uint64()-cursor.Offset)
	addresses := make([]common.Address, size)
	quotes := make([]common.Address, size)
	if size > 0 {
		req := u.rpc.NewRequest().SetContext(ctx).SetBlockNumber(block)
		for i := range addresses {
			req.AddCall(&ethrpc.Call{ABI: factoryABI, Target: Factory, Method: "allLaunches", Params: []any{new(big.Int).SetUint64(cursor.Offset + uint64(i))}}, []any{&addresses[i]})
		}
		if _, err = req.Aggregate(); err != nil {
			return nil, metadata, err
		}
		req = u.rpc.NewRequest().SetContext(ctx).SetBlockNumber(block)
		for i, a := range addresses {
			if a == (common.Address{}) {
				return nil, metadata, ErrState
			}
			req.AddCall(&ethrpc.Call{ABI: tokenABI, Target: hexutil.Encode(a[:]), Method: "quoteAsset"}, []any{&quotes[i]})
		}
		if _, err = req.Aggregate(); err != nil {
			return nil, metadata, err
		}
	}
	pools := make([]entity.Pool, 0, size)
	for i, a := range addresses {
		if quotes[i] == (common.Address{}) {
			return nil, metadata, ErrState
		}
		address := hexutil.Encode(a[:])
		s, _ := json.Marshal(StaticExtra{Factory: Factory, Settlement: Settlement, Quote: hexutil.Encode(quotes[i][:])})
		pools = append(pools, entity.Pool{Address: address, Exchange: u.config.DexID, Type: DexType, Timestamp: int64(head.Time), Tokens: []*entity.PoolToken{{Address: WETH, Swappable: true}, {Address: address, Swappable: true}}, Reserves: entity.PoolReserves{"0", "0"}, StaticExtra: string(s)})
	}
	out, err := json.Marshal(listCursor{Offset: cursor.Offset + size, Block: head.Number.Uint64(), Hash: head.Hash()})
	if err != nil {
		return nil, metadata, err
	}
	return pools, out, nil
}
