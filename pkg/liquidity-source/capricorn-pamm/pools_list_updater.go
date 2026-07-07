package capricornpamm

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	utilabi "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

type PoolsListUpdater struct {
	config         *Config
	ethrpcClient   *ethrpc.Client
	hasInitialized bool
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, client *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{config: cfg, ethrpcClient: client}
}

func (l *PoolsListUpdater) GetNewPools(ctx context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	if l.hasInitialized {
		return nil, nil, nil
	}

	pools := make([]entity.Pool, 0, len(l.config.Pools))
	now := time.Now().Unix()
	for _, raw := range l.config.Pools {
		addr := strings.ToLower(strings.TrimSpace(raw))
		pe, err := l.resolvePool(ctx, addr, now)
		if err != nil {
			logger.WithFields(logger.Fields{"err": err, "pool": addr, "dex": DexType}).
				Errorf("capricorn-pamm: resolve pool")
			return nil, nil, err
		}
		pools = append(pools, pe)
	}
	l.hasInitialized = true
	return pools, nil, nil
}

func (l *PoolsListUpdater) resolvePool(ctx context.Context, addr string, ts int64) (entity.Pool, error) {
	var (
		token0Hx, token1Hx, factoryHx common.Address
		oracleIdRaw                   [32]byte
	)
	req := l.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{ABI: pammPoolABI, Target: addr, Method: methodToken0}, []any{&token0Hx})
	req.AddCall(&ethrpc.Call{ABI: pammPoolABI, Target: addr, Method: methodToken1}, []any{&token1Hx})
	req.AddCall(&ethrpc.Call{ABI: pammPoolABI, Target: addr, Method: methodOracleId}, []any{&oracleIdRaw})
	req.AddCall(&ethrpc.Call{ABI: pammPoolABI, Target: addr, Method: methodFactory}, []any{&factoryHx})
	if _, err := req.Aggregate(); err != nil {
		return entity.Pool{}, err
	}

	token0Addr := strings.ToLower(token0Hx.Hex())
	token1Addr := strings.ToLower(token1Hx.Hex())

	var d0, d1 uint8
	dReq := l.ethrpcClient.NewRequest().SetContext(ctx)
	dReq.AddCall(&ethrpc.Call{ABI: utilabi.Erc20ABI, Target: token0Addr, Method: utilabi.Erc20DecimalsMethod}, []any{&d0})
	dReq.AddCall(&ethrpc.Call{ABI: utilabi.Erc20ABI, Target: token1Addr, Method: utilabi.Erc20DecimalsMethod}, []any{&d1})
	if _, err := dReq.Aggregate(); err != nil {
		return entity.Pool{}, err
	}

	staticExtra, err := json.Marshal(StaticExtra{
		Factory:  strings.ToLower(factoryHx.Hex()),
		OracleId: "0x" + common.Bytes2Hex(oracleIdRaw[:]),
	})
	if err != nil {
		return entity.Pool{}, err
	}

	return entity.Pool{
		Address:  addr,
		Exchange: l.config.DexID,
		Type:     DexType,
		Tokens: []*entity.PoolToken{
			{Address: token0Addr, Decimals: d0, Swappable: true},
			{Address: token1Addr, Decimals: d1, Swappable: true},
		},
		StaticExtra: string(staticExtra),
		Timestamp:   ts,
	}, nil
}
