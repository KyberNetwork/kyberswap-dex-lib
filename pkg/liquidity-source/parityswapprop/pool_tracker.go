package parityswapprop

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{config: cfg, ethrpcClient: ethrpcClient}, nil
}

type poolReserves struct {
	BaseReserve  *big.Int
	QuoteReserve *big.Int
}

// poolParams mirrors PmmPool.Params exactly: feeBps/maxSpreadBps decode as
// uint16, maxStaleness as uint32, and the uint128 notional/floor fields
// decode as *big.Int since Go has no native 128-bit integer.
type poolParams struct {
	FeeBps           uint16
	MaxSpreadBps     uint16
	MaxStaleness     uint32
	MaxSwapNotional  *big.Int
	MaxBlockNotional *big.Int
	MinBaseReserve   *big.Int
	MinQuoteReserve  *big.Int
}

type oraclePrice struct {
	Bid       uint64
	Mid       uint64
	Ask       uint64
	UpdatedAt uint64
}

// GetNewPoolState refreshes every field PoolSimulator.CalcAmountOut needs to
// replicate PmmPool._quote()'s guards -- reserves, params(), oracle(), the
// oracle's own read() (oracle is never hardcoded: it can be rotated per pool
// via setOracle()), paused(), lastSwapBlock(), blockNotional(). The oracle
// multicall is pinned to the first multicall's own resolved block so the
// whole Extra snapshot is internally consistent -- this chain produces
// blocks fast enough (~9/s) that two unpinned calls could easily land on
// different blocks.
func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	var (
		reserves      poolReserves
		params        poolParams
		oracleHx      common.Address
		paused        bool
		lastSwapBlock uint64
		blockNotional *big.Int
	)

	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{ABI: pmmPoolABI, Target: p.Address, Method: methodGetReserves}, []any{&reserves})
	req.AddCall(&ethrpc.Call{ABI: pmmPoolABI, Target: p.Address, Method: methodParams}, []any{&params})
	req.AddCall(&ethrpc.Call{ABI: pmmPoolABI, Target: p.Address, Method: methodOracle}, []any{&oracleHx})
	req.AddCall(&ethrpc.Call{ABI: pmmPoolABI, Target: p.Address, Method: methodPaused}, []any{&paused})
	req.AddCall(&ethrpc.Call{ABI: pmmPoolABI, Target: p.Address, Method: methodLastSwapBlock}, []any{&lastSwapBlock})
	req.AddCall(&ethrpc.Call{ABI: pmmPoolABI, Target: p.Address, Method: methodBlockNotional}, []any{&blockNotional})
	resp, err := req.Aggregate()
	if err != nil {
		return p, err
	}

	oracleAddr := hexutil.Encode(oracleHx[:])
	var price oraclePrice
	oracleReq := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(resp.BlockNumber)
	oracleReq.AddCall(&ethrpc.Call{ABI: pmmOracleABI, Target: oracleAddr, Method: methodOracleRead}, []any{&price})
	if _, err := oracleReq.Aggregate(); err != nil {
		return p, err
	}

	extra, err := json.Marshal(Extra{
		Paused:           paused,
		FeeBps:           params.FeeBps,
		MaxSpreadBps:     params.MaxSpreadBps,
		MaxStaleness:     params.MaxStaleness,
		MaxSwapNotional:  big256.FromBig(params.MaxSwapNotional),
		MaxBlockNotional: big256.FromBig(params.MaxBlockNotional),
		MinBaseReserve:   big256.FromBig(params.MinBaseReserve),
		MinQuoteReserve:  big256.FromBig(params.MinQuoteReserve),
		Oracle:           oracleAddr,
		OracleBid:        price.Bid,
		OracleMid:        price.Mid,
		OracleAsk:        price.Ask,
		OracleUpdatedAt:  price.UpdatedAt,
		LastSwapBlock:    lastSwapBlock,
		BlockNotional:    big256.FromBig(blockNotional),
	})
	if err != nil {
		return p, err
	}

	p.Extra = string(extra)
	p.Reserves = entity.PoolReserves{reserves.BaseReserve.String(), reserves.QuoteReserve.String()}
	p.BlockNumber = resp.BlockNumber.Uint64()
	p.Timestamp = time.Now().Unix()
	return p, nil
}
