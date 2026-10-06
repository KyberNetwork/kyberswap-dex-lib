package uscore

import (
	"context"
	"fmt"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

type PoolsListUpdater struct {
	config *Config
	client *ethrpc.Client
}

func NewPoolsListUpdater(config *Config, client *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{config: config, client: client}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadata []byte) ([]entity.Pool, []byte, error) {
	seen := map[string]bool{}
	if len(metadata) != 0 {
		if err := json.Unmarshal(metadata, &seen); err != nil {
			return nil, metadata, err
		}
		if seen == nil {
			seen = map[string]bool{}
		}
	}
	addresses := make([]string, 0, len(u.config.Pools))
	queued := map[string]bool{}
	for _, raw := range u.config.Pools {
		if !common.IsHexAddress(raw) {
			return nil, metadata, fmt.Errorf("invalid USCore pool: %s", raw)
		}
		addr := common.HexToAddress(raw)
		if addr == (common.Address{}) {
			return nil, metadata, ErrInvalidState
		}
		key := hexutil.Encode(addr[:])
		if !seen[key] && !queued[key] {
			addresses = append(addresses, key)
			queued[key] = true
		}
	}
	if len(addresses) == 0 {
		return nil, metadata, nil
	}
	tokens := make([][2]common.Address, len(addresses))
	req := u.client.R().SetContext(ctx)
	for i, address := range addresses {
		req.AddCall(&ethrpc.Call{ABI: poolABI, Target: address, Method: "quote"}, []any{&tokens[i][0]})
		req.AddCall(&ethrpc.Call{ABI: poolABI, Target: address, Method: "base"}, []any{&tokens[i][1]})
	}
	resp, err := req.TryAggregate()
	if err != nil {
		return nil, metadata, err
	}
	if len(resp.Result) != len(addresses)*2 {
		return nil, metadata, ErrInvalidState
	}
	var pools []entity.Pool
	for i, address := range addresses {
		pair := tokens[i]
		if !resp.Result[2*i] || !resp.Result[2*i+1] || pair[0] == (common.Address{}) || pair[1] == (common.Address{}) || pair[0] == pair[1] {
			continue
		}
		pools = append(pools, entity.Pool{
			Address: address, Exchange: u.config.DexId, Type: DexType, Timestamp: time.Now().Unix(),
			Reserves: entity.PoolReserves{"0", "0"}, Extra: "{}",
			Tokens: []*entity.PoolToken{
				{Address: hexutil.Encode(pair[0][:]), Swappable: true},
				{Address: hexutil.Encode(pair[1][:]), Swappable: true},
			},
		})
		seen[address] = true
	}
	next, err := json.Marshal(seen)
	return pools, next, err
}
