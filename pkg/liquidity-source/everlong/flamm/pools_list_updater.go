package everlongflamm

import (
	"context"
	"math/big"
	"slices"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

// callsPerAggregate bounds one multicall of the lister.
const callsPerAggregate = 200

// PoolsListUpdater lists one entity.Pool per pool of every configured Factory, read on chain: the Factory's
// `pools()`, then each pool's tokens in its own order (token 0 the pool asset). It lists tokens, not trades: which
// pairs trade, and which way, is the tracker's read of the pool's trade mask on every refresh.
type PoolsListUpdater struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{cfg: cfg, ethrpcClient: ethrpcClient}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var md Metadata
	if len(metadataBytes) != 0 && json.Unmarshal(metadataBytes, &md) != nil {
		logger.WithFields(logger.Fields{"dexId": u.cfg.DexID}).Warn("unreadable lister cursor: listing afresh")
	}
	if md.Pools == nil {
		md.Pools = map[string]int{}
	}

	var out []entity.Pool
	for _, f := range u.cfg.Factories {
		factory := common.HexToAddress(f)
		all, counts, err := u.pools(ctx, factory)
		if err != nil { // the other Factories still list; this one again on the next run
			logger.WithFields(logger.Fields{"dexId": u.cfg.DexID, "factory": f, "error": err}).
				Warn("a Factory whose pools did not read")
			continue
		}
		for k, a := range all {
			addr, n := hexutil.Encode(a[:]), counts[k]
			if n <= md.Pools[addr] {
				continue
			}
			if n < 2 || n > maxTokens {
				logger.WithFields(logger.Fields{"dexId": u.cfg.DexID, "pool": addr, "tokens": n}).
					Warn("skipping a pool whose token count this source does not route")
				continue
			}
			tokens, err := u.tokens(ctx, addr, n)
			if err != nil {
				logger.WithFields(logger.Fields{"dexId": u.cfg.DexID, "pool": addr, "error": err}).
					Warn("skipping a pool whose tokens did not read, until the next run")
				continue
			}
			p, err := newEntity(u.cfg, hexutil.Encode(factory[:]), addr, tokens)
			if err != nil {
				return nil, metadataBytes, err
			}
			out = append(out, p)
			md.Pools[addr] = n
		}
	}
	b, err := json.Marshal(md)
	if err != nil {
		return nil, metadataBytes, err
	}
	return out, b, nil
}

// pools reads the Factory's pools and each one's token count, a token added later raising it.
func (u *PoolsListUpdater) pools(ctx context.Context, factory common.Address) ([]common.Address, []int, error) {
	var pools []common.Address
	if _, err := u.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI: factoryABI, Target: hexutil.Encode(factory[:]), Method: "pools",
	}, []any{&pools}).Call(); err != nil {
		return nil, nil, err
	}
	counts := make([]int, len(pools))
	for lo := 0; lo < len(pools); lo += callsPerAggregate {
		chunk := pools[lo:min(lo+callsPerAggregate, len(pools))]
		ns := make([]*big.Int, len(chunk))
		req := u.ethrpcClient.NewRequest().SetContext(ctx)
		for k, a := range chunk {
			req.AddCall(&ethrpc.Call{ABI: poolABI, Target: hexutil.Encode(a[:]), Method: "tokenCount"},
				[]any{&ns[k]})
		}
		res, err := req.TryAggregate()
		if err != nil {
			return nil, nil, err
		}
		for k := range chunk {
			if res.Result[k] && ns[k] != nil && ns[k].IsInt64() {
				counts[lo+k] = int(ns[k].Int64())
			}
		}
	}
	return pools, counts, nil
}

// tokens reads the pool's token list in its own order.
func (u *PoolsListUpdater) tokens(ctx context.Context, pool string, n int) ([]*entity.PoolToken, error) {
	cfgs := make([]tokenConfigResult, n)
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	for t := range n {
		req.AddCall(&ethrpc.Call{ABI: poolABI, Target: pool, Method: "tokenConfig", Params: []any{uint8(t)}},
			[]any{&cfgs[t]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}
	tokens := make([]*entity.PoolToken, n)
	for t, c := range cfgs {
		if c.Config.Token == (common.Address{}) || slices.ContainsFunc(tokens[:t], func(p *entity.PoolToken) bool {
			return p.Address == hexutil.Encode(c.Config.Token[:])
		}) {
			return nil, ErrInvalidToken
		}
		tokens[t] = &entity.PoolToken{Address: hexutil.Encode(c.Config.Token[:]), Swappable: true}
	}
	return tokens, nil
}

func newEntity(cfg *Config, factory, pool string, tokens []*entity.PoolToken) (entity.Pool, error) {
	se, err := json.Marshal(StaticExtra{Factory: factory})
	if err != nil {
		return entity.Pool{}, err
	}
	reserves := make(entity.PoolReserves, len(tokens))
	for k := range reserves {
		reserves[k] = "0"
	}
	return entity.Pool{
		Address:     pool,
		Exchange:    cfg.DexID,
		Type:        DexType,
		Timestamp:   time.Now().Unix(),
		Reserves:    reserves,
		Tokens:      tokens,
		StaticExtra: string(se),
	}, nil
}
