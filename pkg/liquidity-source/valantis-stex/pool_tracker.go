package valantisstex

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (d *PoolTracker) GetNewPoolState(ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return d.getNewPoolState(ctx, p, params, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, pool.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("start getting new state of pool")

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	swapFeeModule := staticExtra.SwapFeeModule.String()

	var (
		reserves struct {
			Token0 *big.Int
			Token1 *big.Int
		}
		effectiveAMMLiquidity *big.Int
		ammState              struct {
			SqrtSpotPriceX96 *big.Int
			SqrtPriceLowX96  *big.Int
			SqrtPriceHighX96 *big.Int
		}
		swapFeeInBipsZtoO, swapFeeInBips0toZ struct {
			Data SwapFeeModuleData
		}
	)
	resp, err := t.ethrpcClient.NewRequest().
		SetContext(ctx).
		SetOverrides(overrides).
		AddCall(&ethrpc.Call{
			ABI:    sovereignPoolABI,
			Target: p.Address,
			Method: "getReserves",
		}, []any{&reserves}).
		AddCall(&ethrpc.Call{
			ABI:    swapFeeModuleABI,
			Target: swapFeeModule,
			Method: "effectiveAMMLiquidity",
		}, []any{&effectiveAMMLiquidity}).
		AddCall(&ethrpc.Call{
			ABI:    swapFeeModuleABI,
			Target: swapFeeModule,
			Method: "getAMMState",
		}, []any{&ammState}).
		AddCall(&ethrpc.Call{
			ABI:    swapFeeModuleABI,
			Target: swapFeeModule,
			Method: "getSwapFeeInBips",
			Params: []any{common.HexToAddress(p.Tokens[0].Address), common.Address{}, bignumber.ZeroBI, common.Address{}, []byte{}},
		}, []any{&swapFeeInBipsZtoO}).
		AddCall(&ethrpc.Call{
			ABI:    swapFeeModuleABI,
			Target: swapFeeModule,
			Method: "getSwapFeeInBips",
			Params: []any{common.HexToAddress(p.Tokens[1].Address), common.Address{}, bignumber.ZeroBI, common.Address{}, []byte{}},
		}, []any{&swapFeeInBips0toZ}).
		Aggregate()
	if err != nil {
		return p, err
	}

	extraBytes, err := json.Marshal(Extra{
		EffectiveAMMLiquidity: uint256.MustFromBig(effectiveAMMLiquidity),
		AMMState: AMMState{
			SqrtSpotPriceX96: uint256.MustFromBig(ammState.SqrtSpotPriceX96),
			SqrtPriceLowX96:  uint256.MustFromBig(ammState.SqrtPriceLowX96),
			SqrtPriceHighX96: uint256.MustFromBig(ammState.SqrtPriceHighX96),
		},
		SwapFeeInBipsZtoO: uint256.MustFromBig(swapFeeInBipsZtoO.Data.FeeInBips),
		SwapFeeInBipsOtoZ: uint256.MustFromBig(swapFeeInBips0toZ.Data.FeeInBips),
	})
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.BlockNumber = resp.BlockNumber.Uint64()
	p.Reserves = entity.PoolReserves{reserves.Token0.String(), reserves.Token1.String()}

	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Info("finish getting new state of pool")

	return p, nil
}
