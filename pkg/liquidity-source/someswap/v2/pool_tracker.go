package someswapv2

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/go-resty/resty/v2"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	v1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/someswap/v1"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
	httpClient   *resty.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) *PoolTracker {
	httpClient := resty.NewWithClient(http.DefaultClient).
		SetBaseURL(config.HTTPConfig.BaseURL).
		SetTimeout(config.HTTPConfig.Timeout.Duration).
		SetRetryCount(config.HTTPConfig.RetryCount)

	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
		httpClient:   httpClient,
	}
}

type DynamicFeeResponse struct {
	Pool           string `json:"pool"`
	BaseFee        uint32 `json:"baseFee"`
	WToken0        string `json:"wToken0"`
	WToken1        string `json:"wToken1"`
	CurrentDynBps  uint32 `json:"currentDynBps"`
	TotalFeeBps    uint32 `json:"totalFeeBps"`
	InBps          uint32 `json:"inBps"`
	OutBps         uint32 `json:"outBps"`
	Config         struct {
		Enabled   bool   `json:"enabled"`
		HalfLife  uint64 `json:"halfLife"`
		MaxCapBps uint32 `json:"maxCapBps"`
	} `json:"config"`
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	reserves, blockNumber, err := d.getReservesFromRPCNode(ctx, p.Address)
	if err != nil {
		return p, err
	}

	if blockNumber != nil && p.BlockNumber > blockNumber.Uint64() {
		return p, nil
	}

	dynBps, err := d.getDynamicFee(ctx, p.Address)
	if err != nil {
		dynBps = 0
	}

	extra := Extra{DynBps: dynBps}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{reserveString(reserves.Reserve0), reserveString(reserves.Reserve1)}
	if blockNumber != nil {
		p.BlockNumber = blockNumber.Uint64()
	}

	return p, nil
}

func (d *PoolTracker) getDynamicFee(ctx context.Context, poolAddress string) (uint32, error) {
	endpoint := fmt.Sprintf("/api/amm/dynamic-fee/%s", poolAddress)

	var result DynamicFeeResponse
	resp, err := d.httpClient.R().
		SetContext(ctx).
		SetResult(&result).
		Get(endpoint)

	if err != nil {
		return 0, fmt.Errorf("failed to call dynamic-fee API: %w", err)
	}

	if !resp.IsSuccess() {
		return 0, fmt.Errorf("dynamic-fee API returned status %v", resp.Status())
	}

	return result.CurrentDynBps, nil
}

func (d *PoolTracker) getReservesFromRPCNode(ctx context.Context, poolAddress string) (v1.ReserveData, *big.Int, error) {
	var reserves v1.ReserveData
	req := d.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    v1.PairABI,
		Target: poolAddress,
		Method: "getReserves",
	}, []any{&reserves})

	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return v1.ReserveData{}, nil, err
	}
	return reserves, resp.BlockNumber, nil
}

func reserveString(reserve *big.Int) string {
	if reserve == nil {
		return "0"
	}
	return reserve.String()
}
