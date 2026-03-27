package liquidcore

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
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

	var (
		reserves struct {
			Reserve0 *big.Int
			Reserve1 *big.Int
		}
		spotPrices struct {
			ForwardPrice uint64
			InversePrice uint64
		}
	)
	resp, err := t.ethrpcClient.R().SetContext(ctx).SetOverrides(overrides).
		AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: p.Address,
			Method: "getReserves",
		}, []any{&reserves}).
		AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: p.Address,
			Method: "getSpotPrices",
		}, []any{&spotPrices}).
		Aggregate()
	if err != nil {
		logger.Errorf("failed to aggregate pool state: %v", err)
		return p, err
	}

	precompile := common.HexToAddress("0x0000000000000000000000000000000000000807")
	oraclePriceBytes, err := t.ethrpcClient.GetETHClient().CallContract(context.Background(), ethereum.CallMsg{
		To:   &precompile,
		Data: common.FromHex("0x000000000000000000000000000000000000000000000000000000000000009f"),
	}, nil)
	if err != nil {
		logger.Errorf("failed to call precompile for oracle price: %v", err)
		return p, err
	}

	extraBytes, err := json.Marshal(Extra{
		SpotPrice:   uint256.NewInt(spotPrices.ForwardPrice),
		OraclePrice: new(uint256.Int).SetBytes(oraclePriceBytes),
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
