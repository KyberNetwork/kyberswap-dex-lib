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
		logger:       logger.WithFields(logger.Fields{"dex": cfg.DexId}),
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	if u.initialized {
		return nil, metadataBytes, nil
	}

	u.logger.Info("start getting new pools")

	pools := make([]entity.Pool, 0, len(u.config.Stex))
	for stex := range u.config.Stex {
		var alm common.Address
		if _, err := u.ethrpcClient.NewRequest().
			SetContext(ctx).
			AddCall(&ethrpc.Call{ABI: stexAMMABI, Target: stex, Method: "pool"}, []any{&alm}).
			Aggregate(); err != nil {
			return nil, nil, err
		}

		var (
			token0, token1, swapFeeModule common.Address
			defaultSwapFeeBips            *big.Int
		)
		if _, err := u.ethrpcClient.NewRequest().
			SetContext(ctx).
			AddCall(&ethrpc.Call{ABI: sovereignPoolABI, Target: alm.String(), Method: "token0"},
				[]any{&token0}).
			AddCall(&ethrpc.Call{ABI: sovereignPoolABI, Target: alm.String(), Method: "token1"},
				[]any{&token1}).
			AddCall(&ethrpc.Call{ABI: sovereignPoolABI, Target: alm.String(), Method: "swapFeeModule"},
				[]any{&swapFeeModule}).
			AddCall(&ethrpc.Call{ABI: sovereignPoolABI, Target: alm.String(), Method: "defaultSwapFeeBips"},
				[]any{&defaultSwapFeeBips}).
			Aggregate(); err != nil {
			return nil, nil, err
		}

		staticExtraBytes, err := json.Marshal(StaticExtra{
			SwapFeeModule:      swapFeeModule,
			DefaultSwapFeeBips: uint256.MustFromBig(defaultSwapFeeBips),
			StexAMM:            common.HexToAddress(stex),
		})
		if err != nil {
			return nil, nil, err
		}

		pools = append(pools, entity.Pool{
			Address:   hexutil.Encode(alm[:]),
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

	u.initialized = true

	return pools, metadataBytes, nil
}
