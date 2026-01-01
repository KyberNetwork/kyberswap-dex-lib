package clear

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	logger.Infof("[Clear] Start getting new state of pool: %v", p.Address)
	if len(p.Tokens) < 2 {
		return entity.Pool{}, ErrPoolNotFound
	}
	// Use a small test amount (1 unit of token0)
	// testAmount := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(p.Tokens[0].Decimals)), nil)
	// var previewResult PreviewSwapResult

	req := d.ethrpcClient.NewRequest().SetContext(ctx)
	previewResult := make(map[int]map[int]*PreviewSwapResult)
	// Test all pairs in BOTH directions (i→j AND j→i)
	// Clear uses oracle pricing, so the rate is not symmetric
	for i := 0; i < len(p.Tokens); i++ {
		for j := 0; j < len(p.Tokens); j++ {
			if i == j {
				continue
			}
			// Initialize previewResult[i] if nil
			if previewResult[i] == nil {
				previewResult[i] = make(map[int]*PreviewSwapResult)
			}
			amountIn := bignumber.TenPowInt(p.Tokens[i].Decimals)
			previewResult[i][j] = &PreviewSwapResult{AmountIn: amountIn}
			req.AddCall(&ethrpc.Call{
				ABI:    clearSwapABI,
				Target: d.config.SwapAddress,
				Method: methodPreviewSwap,
				Params: []any{
					common.HexToAddress(p.Address),
					common.HexToAddress(p.Tokens[i].Address),
					common.HexToAddress(p.Tokens[j].Address),
					amountIn,
				},
			}, []any{previewResult[i][j]})
		}
	}

	if _, err := req.TryAggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("[Clear] failed to call previewSwap")

		return entity.Pool{}, nil
	}
	extra := Extra{
		Reserves: previewResult,
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("[Clear] failed to marshal extra")
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	logger.Infof("[Clear] Finish getting new state of pool: %v", p.Address)

	return p, nil
}
