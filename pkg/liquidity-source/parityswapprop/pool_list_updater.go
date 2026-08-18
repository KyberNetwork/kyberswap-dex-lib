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
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	utilabi "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

// Metadata persists how many entries of registry.getPools() have already
// been resolved into entity.Pool, so a refresh only pays for base()/quote()/
// scale()/decimals() reads on pools added since the last run. getPools()
// generally only grows (PmmRegistry.addPool appends; removePool swap-and-pops,
// but a shrink just means this offset temporarily over-reads, which
// GetNewPools below guards against).
type Metadata struct {
	Offset int `json:"offset"`
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{config: cfg, ethrpcClient: ethrpcClient}
}

// GetNewPools discovers pools via PmmRegistry.getPools(). Reserves are left
// at "0" -- the tracker fills real reserves on the pool's first
// GetNewPoolState refresh.
func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var metadata Metadata
	if len(metadataBytes) != 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return nil, metadataBytes, err
		}
	}

	var poolAddrs []common.Address
	if _, err := u.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: pmmRegistryABI, Target: u.config.Registry, Method: methodGetPools}, []any{&poolAddrs}).
		Call(); err != nil {
		return nil, metadataBytes, err
	}

	if metadata.Offset >= len(poolAddrs) {
		return nil, metadataBytes, nil
	}
	newAddrs := poolAddrs[metadata.Offset:]

	pools, err := u.resolvePools(ctx, newAddrs)
	if err != nil {
		return nil, metadataBytes, err
	}

	newMetadataBytes, err := json.Marshal(Metadata{Offset: metadata.Offset + len(newAddrs)})
	if err != nil {
		return nil, metadataBytes, err
	}

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) resolvePools(ctx context.Context, addrs []common.Address) ([]entity.Pool, error) {
	n := len(addrs)
	baseHx := make([]common.Address, n)
	quoteHx := make([]common.Address, n)
	baseScaleRaw := make([]*big.Int, n)
	quoteScaleRaw := make([]*big.Int, n)

	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i, addr := range addrs {
		target := hexutil.Encode(addr[:])
		req.AddCall(&ethrpc.Call{ABI: PmmPoolABI, Target: target, Method: methodBase}, []any{&baseHx[i]})
		req.AddCall(&ethrpc.Call{ABI: PmmPoolABI, Target: target, Method: methodQuote}, []any{&quoteHx[i]})
		req.AddCall(&ethrpc.Call{ABI: PmmPoolABI, Target: target, Method: methodBaseScale}, []any{&baseScaleRaw[i]})
		req.AddCall(&ethrpc.Call{ABI: PmmPoolABI, Target: target, Method: methodQuoteScale}, []any{&quoteScaleRaw[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}

	baseAddr := make([]string, n)
	quoteAddr := make([]string, n)
	decReq := u.ethrpcClient.NewRequest().SetContext(ctx)
	baseDec := make([]uint8, n)
	quoteDec := make([]uint8, n)
	for i := range addrs {
		baseAddr[i] = hexutil.Encode(baseHx[i][:])
		quoteAddr[i] = hexutil.Encode(quoteHx[i][:])
		decReq.AddCall(&ethrpc.Call{ABI: utilabi.Erc20ABI, Target: baseAddr[i], Method: utilabi.Erc20DecimalsMethod}, []any{&baseDec[i]})
		decReq.AddCall(&ethrpc.Call{ABI: utilabi.Erc20ABI, Target: quoteAddr[i], Method: utilabi.Erc20DecimalsMethod}, []any{&quoteDec[i]})
	}
	if _, err := decReq.Aggregate(); err != nil {
		return nil, err
	}

	pools := make([]entity.Pool, 0, n)
	now := time.Now().Unix()
	for i, addr := range addrs {
		staticExtra, err := json.Marshal(StaticExtra{
			Base:       baseAddr[i],
			Quote:      quoteAddr[i],
			BaseScale:  baseScaleRaw[i].String(),
			QuoteScale: quoteScaleRaw[i].String(),
		})
		if err != nil {
			return nil, err
		}

		pools = append(pools, entity.Pool{
			Address:     hexutil.Encode(addr[:]),
			Exchange:    u.config.DexID,
			Type:        DexType,
			Timestamp:   now,
			Reserves:    entity.PoolReserves{"0", "0"},
			Tokens:      []*entity.PoolToken{{Address: baseAddr[i], Decimals: baseDec[i], Swappable: true}, {Address: quoteAddr[i], Decimals: quoteDec[i], Swappable: true}},
			StaticExtra: string(staticExtra),
		})
	}
	return pools, nil
}
