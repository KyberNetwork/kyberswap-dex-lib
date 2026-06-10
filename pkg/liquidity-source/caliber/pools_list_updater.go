package caliber

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

	contract := strings.ToLower(strings.TrimSpace(l.config.Contract))
	okList := l.verifyPairs(ctx, contract)

	now := time.Now().Unix()
	pools := make([]entity.Pool, 0, len(l.config.Pairs))
	for i, pc := range l.config.Pairs {
		pairID := strings.ToLower(strings.TrimSpace(pc.PairID))
		token0 := strings.ToLower(strings.TrimSpace(pc.Token0))
		token1 := strings.ToLower(strings.TrimSpace(pc.Token1))

		if okList != nil && !okList[i] {
			logger.WithFields(logger.Fields{
				"dex": DexType, "contract": contract, "pairId": pairID,
				"token0": token0, "token1": token1,
			}).Error("pair validation failed, skipping")
			continue
		}

		staticExtra, err := json.Marshal(StaticExtra{Contract: contract, PairID: pairID})
		if err != nil {
			return nil, nil, err
		}

		pools = append(pools, entity.Pool{
			Address:  pairID,
			Exchange: l.config.DexID,
			Type:     DexType,
			Reserves: entity.PoolReserves{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: token0, Swappable: true},
				{Address: token1, Swappable: true},
			},
			Extra:       "{}",
			StaticExtra: string(staticExtra),
			Timestamp:   now,
		})
	}

	l.hasInitialized = true
	return pools, nil, nil
}

func (l *PoolsListUpdater) verifyPairs(ctx context.Context, contract string) []bool {
	if l.ethrpcClient == nil || len(l.config.Pairs) == 0 {
		return nil
	}

	onchain := make([][32]byte, len(l.config.Pairs))
	req := l.ethrpcClient.NewRequest().SetContext(ctx)
	for i, pc := range l.config.Pairs {
		req.AddCall(&ethrpc.Call{
			ABI:    caliberABI,
			Target: contract,
			Method: methodGetPairId,
			Params: []any{
				common.HexToAddress(pc.Token0),
				common.HexToAddress(pc.Token1),
			},
		}, []any{&onchain[i]})
	}

	resp, err := req.TryAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{"dex": DexType, "err": err}).
			Warn("pair verification call failed")
		return nil
	}

	verified := make([]bool, len(l.config.Pairs))
	for i, pc := range l.config.Pairs {
		if i < len(resp.Result) && resp.Result[i] {
			want := common.HexToHash(strings.TrimSpace(pc.PairID))
			verified[i] = common.Hash(onchain[i]) == want
		}
	}
	return verified
}
