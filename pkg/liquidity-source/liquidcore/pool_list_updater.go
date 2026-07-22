package liquidcore

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/rs/zerolog/log"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

type (
	PoolsListUpdater struct {
		config       *Config
		ethrpcClient *ethrpc.Client
	}
)

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	l := log.Ctx(ctx).With().Str("dexID", u.config.DexId).Logger()
	l.Info().Msg("start getting new pools")

	var metadata Metadata
	if len(metadataBytes) != 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return nil, metadataBytes, err
		}
	}

	var poolAddrs []common.Address
	if _, err := u.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: routerABI, Target: u.config.Router, Method: "getPools"}, []any{&poolAddrs}).
		Call(); err != nil {
		return nil, nil, err
	}

	var poolsChecksum common.Address
	for _, pool := range poolAddrs {
		for i := range common.AddressLength {
			poolsChecksum[i] ^= pool[i]
		}
	}
	if metadata.LastCount == len(poolAddrs) && metadata.LastPoolsChecksum == poolsChecksum {
		return nil, metadataBytes, nil
	}
	metadata.LastCount, metadata.LastPoolsChecksum = len(poolAddrs), poolsChecksum

	pools := make([]entity.Pool, 0, len(poolAddrs))
	if len(poolAddrs) > 0 {
		tokenResps := make([]struct{ Token0, Token1 common.Address }, len(poolAddrs))
		req := u.ethrpcClient.NewRequest().SetContext(ctx)
		for i, poolAddr := range poolAddrs {
			req.AddCall(&ethrpc.Call{
				ABI:    poolABI,
				Target: hexutil.Encode(poolAddr[:]),
				Method: "getTokens",
			}, []any{&tokenResps[i]})
		}
		// TryAggregate (not Call/Aggregate) so one dead/reverting pool
		// address from the router's getPools() list doesn't abort discovery
		// of every other pool -- and, since GetNewPools only persists
		// metadataBytes on a fully successful return, doesn't get every
		// subsequent run stuck retrying the same broken address forever.
		resp, err := req.TryAggregate()
		if err != nil {
			return nil, nil, err
		}

		for i, poolAddr := range poolAddrs {
			if !resp.Result[i] {
				l.Warn().Str("pool", hexutil.Encode(poolAddr[:])).Msg("skipping pool: getTokens call failed")
				continue
			}

			pools = append(pools, entity.Pool{
				Address:   hexutil.Encode(poolAddr[:]),
				Exchange:  u.config.DexId,
				Type:      DexType,
				Timestamp: time.Now().Unix(),
				Reserves:  []string{"0", "0"},
				Tokens: []*entity.PoolToken{
					{Address: hexutil.Encode(tokenResps[i].Token0[:]), Swappable: true},
					{Address: hexutil.Encode(tokenResps[i].Token1[:]), Swappable: true},
				},
				Extra: "{}",
			})
		}
	}

	l.Info().Int("count", len(pools)).Msg("finished getting new pools")

	metadataBytes, _ = json.Marshal(metadata)
	return pools, metadataBytes, nil
}
