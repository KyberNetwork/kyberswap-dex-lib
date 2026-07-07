package miromigrator

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolListUpdater struct {
	cfg            *Config
	client         *ethrpc.Client
	hasInitialized bool
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolListUpdater)

func NewPoolListUpdater(
	cfg *Config,
	client *ethrpc.Client,
) *PoolListUpdater {
	return &PoolListUpdater{
		cfg:    cfg,
		client: client,
	}
}

func (u *PoolListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	if u.hasInitialized {
		return nil, nil, nil
	}
	u.hasInitialized = true

	extra, blockNumber, err := getPoolExtra(ctx, u.cfg, u.client)
	if err != nil {
		return nil, nil, err
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return nil, nil, err
	}

	tokenAddresses := []string{u.cfg.PSP}
	if u.cfg.SePSP1 != "" {
		tokenAddresses = append(tokenAddresses, u.cfg.SePSP1)
	}
	tokenAddresses = append(tokenAddresses, u.cfg.VLR)

	tokens := make([]*entity.PoolToken, len(tokenAddresses))
	reserves := make([]string, len(tokenAddresses))
	for i, addr := range tokenAddresses {
		tokens[i] = &entity.PoolToken{
			Address:   strings.ToLower(addr),
			Swappable: true,
		}
		reserves[i] = defaultReserve
	}

	return []entity.Pool{
		{
			Address:     strings.ToLower(u.cfg.Migrator),
			Exchange:    string(valueobject.ExchangeMiroMigrator),
			Type:        DexType,
			Timestamp:   time.Now().Unix(),
			Reserves:    reserves,
			Tokens:      tokens,
			BlockNumber: blockNumber,
			Extra:       string(extraBytes),
		},
	}, nil, nil
}

func getPoolExtra(ctx context.Context, cfg *Config, ethrpcClient *ethrpc.Client) (string, uint64, error) {
	var extra PoolExtra
	req := ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    MigratorABI,
		Target: cfg.Migrator,
		Method: "paused",
	}, []any{&extra.Paused})

	resp, err := req.Aggregate()
	if err != nil {
		return "", 0, err
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return "", 0, err
	}

	return string(extraBytes), resp.BlockNumber.Uint64(), nil
}
