package liquidcore

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
	logger.Infof("getting new state for pool %v", p.Address)
	defer logger.Infof("finished getting new state for pool %v", p.Address)

	token0 := common.HexToAddress(p.Tokens[0].Address)
	token1 := common.HexToAddress(p.Tokens[1].Address)

	decimal0 := p.Tokens[0].Decimals
	decimal1 := p.Tokens[1].Decimals

	var (
		reserves struct {
			Reserve0 *big.Int
			Reserve1 *big.Int
		}
		spotPrices struct {
			ForwardPrice uint64 // token0/token1 price
			InversePrice uint64 // token1/token0 price
		}
		poolFees struct {
			FeeToken0In *big.Int
			FeeToken1In *big.Int
		}
		rate01 *big.Int
		rate10 *big.Int
	)
	resp, err := t.ethrpcClient.R().SetContext(ctx).SetOverrides(overrides).
		AddCall(&ethrpc.Call{
			ABI:    PoolABI,
			Target: p.Address,
			Method: "getReserves",
		}, []any{&reserves}).
		AddCall(&ethrpc.Call{
			ABI:    PoolABI,
			Target: p.Address,
			Method: "getSpotPrices",
		}, []any{&spotPrices}).
		AddCall(&ethrpc.Call{
			ABI:    PoolABI,
			Target: p.Address,
			Method: "getPoolFees",
		}, []any{&poolFees}).
		AddCall(&ethrpc.Call{
			ABI:    PoolABI,
			Target: p.Address,
			Method: "estimateSwap",
			Params: []any{token0, token1, bignumber.TenPowInt(decimal0)},
		}, []any{&rate01}).
		AddCall(&ethrpc.Call{
			ABI:    PoolABI,
			Target: p.Address,
			Method: "estimateSwap",
			Params: []any{token1, token0, bignumber.TenPowInt(decimal1)},
		}, []any{&rate10}).
		Aggregate()
	if err != nil {
		logger.Errorf("failed to aggregate pool state: %v", err)
		return p, err
	}

	extraBytes, err := json.Marshal(Extra{
		ForwardPrice: uint256.NewInt(spotPrices.ForwardPrice),
		InversePrice: uint256.NewInt(spotPrices.InversePrice),
		FeeToken0In:  uint256.MustFromBig(poolFees.FeeToken0In),
		FeeToken1In:  uint256.MustFromBig(poolFees.FeeToken1In),
		Rate01:       uint256.MustFromBig(rate01),
		Rate10:       uint256.MustFromBig(rate10),
	})
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.BlockNumber = resp.BlockNumber.Uint64()
	p.Reserves = entity.PoolReserves{reserves.Reserve0.String(), reserves.Reserve1.String()}

	return p, nil
}
