package rsweth

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/swell/common"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolListUpdater struct {
	ethrpcClient *ethrpc.Client

	hasInitialized bool
}

var _ = poollist.RegisterFactoryE(DexType, NewPoolListUpdater)

func NewPoolListUpdater(
	ethrpcClient *ethrpc.Client,
) *PoolListUpdater {
	return &PoolListUpdater{
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	if u.hasInitialized {
		return nil, nil, nil
	}

	u.hasInitialized = true

	extra, blockNumber, err := u.getExtra(ctx)
	if err != nil {
		return nil, nil, err
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return nil, nil, err
	}

	return []entity.Pool{
		{
			Address:   strings.ToLower(common.RSWETH),
			Exchange:  string(valueobject.ExchangeSwellRSWETH),
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{reserves, reserves},
			Tokens: []*entity.PoolToken{
				{
					Address:   strings.ToLower(common.WETH),
					Symbol:    "WETH",
					Decimals:  18,
					Name:      "Wrapped Ether",
					Swappable: true,
				},
				{
					Address:   strings.ToLower(common.RSWETH),
					Symbol:    "rswETH",
					Decimals:  18,
					Name:      "rswETH",
					Swappable: true,
				},
			},
			BlockNumber: blockNumber,
			Extra:       string(extraBytes),
		},
	}, nil, nil
}

func (u *PoolListUpdater) getExtra(ctx context.Context) (PoolExtra, uint64, error) {
	var (
		paused          bool
		ethToRswETHRate *big.Int
	)

	getPoolStateRequest := u.ethrpcClient.NewRequest().SetContext(ctx)

	getPoolStateRequest.AddCall(&ethrpc.Call{
		ABI:    common.AccessControlManagerABI,
		Target: common.AccessControlManager,
		Method: common.AccessControlManagerMethodCoreMethodsPaused,
		Params: []interface{}{},
	}, []interface{}{&paused})

	getPoolStateRequest.AddCall(&ethrpc.Call{
		ABI:    common.RSWETHABI,
		Target: common.RSWETH,
		Method: common.RSWETHMethodETHTORSWETHRate,
		Params: []interface{}{},
	}, []interface{}{&ethToRswETHRate})

	resp, err := getPoolStateRequest.TryAggregate()
	if err != nil {
		return PoolExtra{}, 0, err
	}
	if resp.BlockNumber == nil {
		resp.BlockNumber = big.NewInt(0)
	}

	return PoolExtra{
		Paused:          paused,
		ETHToRswETHRate: ethToRswETHRate,
	}, resp.BlockNumber.Uint64(), nil
}
