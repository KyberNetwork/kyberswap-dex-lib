package bouncetech

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
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params, nil)
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
	logger.WithFields(logger.Fields{"address": p.Address}).Info("[bounce-tech] getting pool state")

	ltAddr := p.Address

	var (
		exchangeRate   = new(big.Int)
		targetLeverage = new(big.Int)
		mintPaused     bool
		totalSupply    = new(big.Int)
		baseBalance    = new(big.Int)
		redemptionFee  = new(big.Int)
		minTxSize      = new(big.Int)
	)

	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}

	req.AddCall(&ethrpc.Call{
		ABI:    leveragedTokenABI,
		Target: ltAddr,
		Method: "exchangeRate",
	}, []any{&exchangeRate}).
		AddCall(&ethrpc.Call{
			ABI:    leveragedTokenABI,
			Target: ltAddr,
			Method: "targetLeverage",
		}, []any{&targetLeverage}).
		AddCall(&ethrpc.Call{
			ABI:    leveragedTokenABI,
			Target: ltAddr,
			Method: "mintPaused",
		}, []any{&mintPaused}).
		AddCall(&ethrpc.Call{
			ABI:    leveragedTokenABI,
			Target: ltAddr,
			Method: "totalSupply",
		}, []any{&totalSupply}).
		AddCall(&ethrpc.Call{
			ABI:    leveragedTokenABI,
			Target: ltAddr,
			Method: "baseAssetBalance",
		}, []any{&baseBalance}).
		AddCall(&ethrpc.Call{
			ABI:    globalStorageABI,
			Target: t.config.GlobalStorageAddress,
			Method: "redemptionFee",
		}, []any{&redemptionFee}).
		AddCall(&ethrpc.Call{
			ABI:    globalStorageABI,
			Target: t.config.GlobalStorageAddress,
			Method: "minTransactionSize",
		}, []any{&minTxSize})

	resp, err := req.Aggregate()
	if err != nil {
		return p, err
	}

	extra := Extra{
		ExchangeRate:       uint256.MustFromBig(exchangeRate),
		RedemptionFee:      uint256.MustFromBig(redemptionFee),
		TargetLeverage:     uint256.MustFromBig(targetLeverage),
		MinTransactionSize: uint256.MustFromBig(minTxSize),
		MintPaused:         mintPaused,
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.BlockNumber = resp.BlockNumber.Uint64()
	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{
		baseBalance.String(),
		totalSupply.String(),
	}

	return p, nil
}
