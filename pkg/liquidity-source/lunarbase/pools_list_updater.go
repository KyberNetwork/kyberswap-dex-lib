package lunarbase

import (
	"context"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolsListUpdater struct {
	config         *Config
	ethrpcClient   *ethrpc.Client
	hasInitialized bool
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(config *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	if config.DexID == "" {
		config.DexID = DexType
	}
	if config.ChainID == 0 {
		config.ChainID = valueobject.ChainIDBase
	}

	return &PoolsListUpdater{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	if u.hasInitialized {
		return nil, metadataBytes, nil
	}

	poolEntities := make([]entity.Pool, 0, len(u.config.Pools))
	poolAddrs := make([]common.Address, 0, len(u.config.Pools))
	for _, poolAddr := range u.config.Pools {
		poolAddr = strings.ToLower(poolAddr)
		state, err := fetchRPCState(ctx, poolAddr, u.config.ChainID, u.ethrpcClient, nil)
		if err != nil {
			return nil, metadataBytes, err
		}

		staticExtraBytes, _ := json.Marshal(StaticExtra{
			HasNative: state.hasNative,
		})
		poolEntity := &entity.Pool{
			Address:  poolAddr,
			Exchange: u.config.DexID,
			Type:     DexType,
			Tokens: []*entity.PoolToken{
				{Address: state.tokenX, Swappable: true},
				{Address: state.tokenY, Swappable: true},
			},
			StaticExtra: string(staticExtraBytes),
		}
		poolEntity, err = buildEntityPool(poolEntity, state)
		if err != nil {
			continue
		}

		poolEntities = append(poolEntities, *poolEntity)
		poolAddrs = append(poolAddrs, common.HexToAddress(poolAddr))
	}

	if u.config.WsURL != "" || u.config.FlashWsURL != "" {
		InitFlashBlockSubscriber(
			u.config.WsURL,
			u.config.FlashWsURL,
			poolAddrs,
		)
	}

	u.hasInitialized = true

	return poolEntities, metadataBytes, nil
}
