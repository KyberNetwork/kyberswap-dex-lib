package prop

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

type poolsListUpdaterMetadata struct {
	Initialized bool `json:"initialized"`
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	log := logger.WithFields(logger.Fields{"dexId": u.cfg.DexID})

	var meta poolsListUpdaterMetadata
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &meta); err != nil {
			log.Warnf("1010-prop: failed to parse metadata: %v", err)
		}
	}
	if meta.Initialized {
		return nil, metadataBytes, nil
	}

	log.Info("1010-prop: start get pools")

	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	var assets []common.Address
	req.AddCall(&ethrpc.Call{
		ABI:    routerABI,
		Target: u.cfg.RouterAddress,
		Method: "getAssets",
		Params: nil,
	}, []any{&assets})

	if _, err := req.TryAggregate(); err != nil {
		log.Errorf("1010-prop: getAssets failed: %v", err)
		return nil, metadataBytes, err
	}

	routerAddr := common.HexToAddress(u.cfg.RouterAddress)
	staticExtraBytes, _ := json.Marshal(StaticExtra{
		RouterAddress: strings.ToLower(u.cfg.RouterAddress),
	})

	now := time.Now().Unix()
	pools := make([]entity.Pool, 0, len(assets)*(len(assets)-1)/2)

	for i := range assets {
		for j := i + 1; j < len(assets); j++ {
			poolAddr := pairPoolAddress(routerAddr, assets[i], assets[j])
			p := entity.Pool{
				Address:   poolAddr,
				Exchange:  u.cfg.DexID,
				Type:      DexType,
				Timestamp: now,
				Reserves:  entity.PoolReserves{"0", "0"},
				Tokens: []*entity.PoolToken{
					{Address: strings.ToLower(hexutil.Encode(assets[i][:])), Swappable: true},
					{Address: strings.ToLower(hexutil.Encode(assets[j][:])), Swappable: true},
				},
				Extra:       "{}",
				StaticExtra: string(staticExtraBytes),
			}
			pools = append(pools, p)
		}
	}

	newMeta, _ := json.Marshal(poolsListUpdaterMetadata{Initialized: true})
	return pools, newMeta, nil
}

// pairPoolAddress derives a deterministic pool address from the router and the two token addresses.
func pairPoolAddress(router, token0, token1 common.Address) string {
	hash := crypto.Keccak256(router.Bytes(), token0.Bytes(), token1.Bytes())
	return strings.ToLower(common.BytesToAddress(hash).Hex())
}
