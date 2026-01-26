package valantisstex

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
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
	logger.Infof("getting new state for pool %v", p.Address)
	defer logger.Infof("finished getting new state for pool %v", p.Address)

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	swapFeeModule := hexutil.Encode(staticExtra.SwapFeeModule[:])
	stexAMM := hexutil.Encode(staticExtra.StexAMM[:])

	var (
		reserves struct {
			Token0 *big.Int
			Token1 *big.Int
		}
		withdrawalModule                     common.Address
		rate0To1                             *big.Int
		rate1To0                             *big.Int
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
			Method: "getSwapFeeInBips",
			Params: []any{common.HexToAddress(p.Tokens[0].Address), common.Address{}, bignumber.ZeroBI, common.Address{}, []byte{}},
		}, []any{&swapFeeInBipsZtoO}).
		AddCall(&ethrpc.Call{
			ABI:    swapFeeModuleABI,
			Target: swapFeeModule,
			Method: "getSwapFeeInBips",
			Params: []any{common.HexToAddress(p.Tokens[1].Address), common.Address{}, bignumber.ZeroBI, common.Address{}, []byte{}},
		}, []any{&swapFeeInBips0toZ}).
		AddCall(&ethrpc.Call{
			ABI:    stexAMMABI,
			Target: stexAMM,
			Method: "withdrawalModule",
		}, []any{&withdrawalModule}).
		Aggregate()
	if err != nil {
		return p, err
	}

	_, err = t.ethrpcClient.NewRequest().
		SetContext(ctx).
		SetOverrides(overrides).
		SetBlockNumber(resp.BlockNumber).
		AddCall(&ethrpc.Call{
			ABI:    withdrawalModuleABI,
			Target: withdrawalModule.String(),
			Method: "convertToToken0",
			Params: []any{bignumber.BONE},
		}, []any{&rate1To0}).
		AddCall(&ethrpc.Call{
			ABI:    withdrawalModuleABI,
			Target: withdrawalModule.String(),
			Method: "convertToToken1",
			Params: []any{bignumber.BONE},
		}, []any{&rate0To1}).
		Aggregate()
	if err != nil {
		return p, err
	}

	cfg, exists := t.config.Stex[stexAMM]
	if !exists {
		logger.Errorf("gas config not found for stex: %s", stexAMM)
		return p, nil
	}

	extraBytes, err := json.Marshal(Extra{
		WithdrawalModule:  withdrawalModule,
		SwapFeeInBipsZtoO: uint256.MustFromBig(swapFeeInBipsZtoO.Data.FeeInBips),
		SwapFeeInBipsOtoZ: uint256.MustFromBig(swapFeeInBips0toZ.Data.FeeInBips),
		Rate0To1:          uint256.MustFromBig(rate0To1),
		Rate1To0:          uint256.MustFromBig(rate1To0),
		Gas:               cfg.Gas,
	})
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.BlockNumber = resp.BlockNumber.Uint64()
	p.Reserves = entity.PoolReserves{reserves.Token0.String(), reserves.Token1.String()}

	return p, nil
}
