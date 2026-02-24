package liquidcore

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

type (
	PoolsListUpdater struct {
		config       *Config
		ethrpcClient *ethrpc.Client
		initialized  bool
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
	if u.initialized {
		return nil, metadataBytes, nil
	}

	logger.Info("start getting new pools")

	pools := make([]entity.Pool, 0, len(u.config.Pools))
	for _, address := range u.config.Pools {
		var tokenResp struct {
			Token0, Token1 common.Address
		}

		if _, err := u.ethrpcClient.NewRequest().SetContext(ctx).
			AddCall(&ethrpc.Call{ABI: PoolABI, Target: address, Method: "getTokens"}, []any{&tokenResp}).
			Call(); err != nil {
			return nil, nil, err
		}

		pools = append(pools, entity.Pool{
			Address:   strings.ToLower(address),
			Exchange:  u.config.DexId,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: hexutil.Encode(tokenResp.Token0[:]), Swappable: true},
				{Address: hexutil.Encode(tokenResp.Token1[:]), Swappable: true},
			},
			Extra: "{}",
		})

	}

	logger.Infof("finish getting new pools, got %d pools", len(pools))

	u.initialized = true

	return pools, metadataBytes, nil
}
