package valantisstex

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

type (
	PoolsListUpdater struct {
		config       *Config
		ethrpcClient *ethrpc.Client
		logger       logger.Logger
	}
)

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
		logger:       logger.WithFields(logger.Fields{"dex": cfg.DexId}),
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	u.logger.Info("start getting new pools")

	pools := make([]entity.Pool, 0, len(u.config.SovereignPools))
	for _, p := range u.config.SovereignPools {
		var (
			token0, token1, swapFeeModule common.Address
			defaultSwapFeeBips            *big.Int
		)
		if _, err := u.ethrpcClient.NewRequest().
			SetContext(ctx).
			AddCall(&ethrpc.Call{ABI: sovereignPoolABI, Target: p.String(), Method: "token0"},
				[]any{&token0}).
			AddCall(&ethrpc.Call{ABI: sovereignPoolABI, Target: p.String(), Method: "token1"},
				[]any{&token1}).
			AddCall(&ethrpc.Call{ABI: sovereignPoolABI, Target: p.String(), Method: "swapFeeModule"},
				[]any{&swapFeeModule}).
			AddCall(&ethrpc.Call{ABI: sovereignPoolABI, Target: p.String(), Method: "defaultSwapFeeBips"},
				[]any{&defaultSwapFeeBips}).
			Aggregate(); err != nil {
			return nil, nil, err
		}

		staticExtraBytes, err := json.Marshal(StaticExtra{
			SwapFeeModule:      swapFeeModule,
			DefaultSwapFeeBips: uint256.MustFromBig(defaultSwapFeeBips),
		})
		if err != nil {
			return nil, nil, err
		}

		pools = append(pools, entity.Pool{
			Address:   hexutil.Encode(p[:]),
			Exchange:  u.config.DexId,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{"0", "0"},
			Tokens: []*entity.PoolToken{
				{
					Address:   hexutil.Encode(token0[:]),
					Swappable: true,
				},
				{
					Address:   hexutil.Encode(token1[:]),
					Swappable: true,
				},
			},
			StaticExtra: string(staticExtraBytes),
		})
	}

	u.logger.Infof("finish getting new pools, got %d pools", len(pools))

	return pools, metadataBytes, nil
}
