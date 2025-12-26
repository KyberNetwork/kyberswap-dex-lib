package tessera

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type PoolsListUpdater struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	u.logger().Info("Start getting new pools.")

	// 1. Get pairs from Indexer
	var pairs [][]common.Address
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    TesseraIndexerABI,
		Target: u.cfg.IndexerAddr,
		Method: "getTesseraPairs",
	}, []any{&pairs})

	u.logger().Infof("Calling Indexer at %s", u.cfg.IndexerAddr)
	if _, err := req.Call(); err != nil {
		u.logger().Errorf("Indexer call failed: %v", err)
		return nil, nil, err
	}
	u.logger().Infof("Found %d pairs from Indexer", len(pairs))

	// 2. Get pool addresses from Engine for each pair
	type getPoolResp struct {
		Exists bool           `abi:"exists"`
		Pool   common.Address `abi:"pool"`
	}

	var pools []entity.Pool
	engineReq := u.ethrpcClient.NewRequest().SetContext(ctx)
	resps := make([]getPoolResp, len(pairs))

	for i, pair := range pairs {
		token0 := pair[0]
		token1 := pair[1]
		engineReq.AddCall(&ethrpc.Call{
			ABI:    TesseraEngineABI,
			Target: u.cfg.EngineAddr,
			Method: "getTesseraPool",
			Params: []any{token0, token1},
		}, []any{&resps[i]})
	}

	u.logger().Infof("Calling Engine for %d pools at %s", len(pairs), u.cfg.EngineAddr)
	// Use TryAggregate which worked before (gave unmarshal error, which means call reached contract)
	resp, err := engineReq.TryAggregate()
	if err != nil {
		u.logger().Errorf("Engine call failed: %v", err)
		return nil, nil, err
	}

	for i, r := range resps {
		if !resp.Result[i] || !r.Exists || r.Pool == (common.Address{}) {
			u.logger().Debugf("Skipping idx %d: success=%v, exists=%v, addr=%s", i, resp.Result[i], r.Exists, r.Pool.Hex())
			continue
		}

		u.logger().Infof("Found pool: %s for pair [%s, %s]", r.Pool.Hex(), pairs[i][0].Hex(), pairs[i][1].Hex())

		tokens := []*entity.PoolToken{
			{Address: strings.ToLower(pairs[i][0].Hex()), Swappable: true},
			{Address: strings.ToLower(pairs[i][1].Hex()), Swappable: true},
		}

		staticExtraBytes, _ := json.Marshal(StaticExtra{
			BaseToken:  strings.ToLower(pairs[i][0].Hex()),
			QuoteToken: strings.ToLower(pairs[i][1].Hex()),
			EngineAddr: u.cfg.EngineAddr,
		})

		p := entity.Pool{
			Address:     strings.ToLower(r.Pool.Hex()),
			Exchange:    u.cfg.DexId,
			Type:        "tessera",
			Timestamp:   time.Now().Unix(),
			Reserves:    entity.PoolReserves{"0", "0"},
			Tokens:      tokens,
			StaticExtra: string(staticExtraBytes),
		}
		pools = append(pools, p)
	}

	u.logger().Infof("Total valid pools found: %d", len(pools))
	return pools, nil, nil
}

func (u *PoolsListUpdater) logger() logger.Logger {
	return logger.WithFields(logger.Fields{
		"dexId": u.cfg.DexId,
	})
}
