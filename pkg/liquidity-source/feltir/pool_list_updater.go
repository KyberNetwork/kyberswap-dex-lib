package feltir

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdaterMetadata struct {
	Offset int `json:"offset"`
}

type PoolsListUpdater struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{cfg: cfg, ethrpcClient: ethrpcClient}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	log := logger.WithFields(logger.Fields{"dexId": u.cfg.DexID})
	log.Info("feltir: start get pools")

	var metadata PoolsListUpdaterMetadata
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			log.Warnf("feltir: parse metadata failed: %v", err)
		}
	}

	var pools []PoolRPC
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    feltirABI,
		Target: u.cfg.FeltirAddress,
		Method: "getPools",
	}, []any{&pools})

	if _, err := req.Call(); err != nil {
		log.Errorf("feltir: getPools failed: %v", err)
		return nil, metadataBytes, err
	}

	if metadata.Offset >= len(pools) {
		return nil, metadataBytes, nil
	}

	newPools := pools[metadata.Offset:]

	staticExtraBytes, _ := json.Marshal(StaticExtra{
		FeltirAddress: strings.ToLower(u.cfg.FeltirAddress),
	})

	now := time.Now().Unix()

	result := make([]entity.Pool, 0, len(newPools))
	for _, p := range newPools {
		token0 := strings.ToLower(p.Token0.Hex())
		token1 := strings.ToLower(p.Token1.Hex())

		result = append(result, entity.Pool{
			Address:   u.cfg.DexID + "_" + token0 + "_" + token1,
			Exchange:  u.cfg.DexID,
			Type:      DexType,
			Timestamp: now,
			Reserves:  entity.PoolReserves{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: token0, Swappable: true},
				{Address: token1, Swappable: true},
			},
			Extra:       "{}",
			StaticExtra: string(staticExtraBytes),
		})
	}

	newMeta, err := json.Marshal(PoolsListUpdaterMetadata{Offset: len(pools)})
	if err != nil {
		return result, metadataBytes, nil
	}

	return result, newMeta, nil
}
