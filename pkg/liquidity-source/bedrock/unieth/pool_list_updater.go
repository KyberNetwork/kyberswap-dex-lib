package unieth

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolListUpdater struct {
	ethrpcClient *ethrpc.Client

	hasInitialized bool
}

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
			Address:   strings.ToLower(Staking),
			Exchange:  string(valueobject.ExchangeBedrockUniETH),
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{reserves, reserves},
			Tokens: []*entity.PoolToken{
				{
					Address:   strings.ToLower(WETH),
					Symbol:    "WETH",
					Decimals:  18,
					Name:      "Wrapped Ether",
					Swappable: true,
				},
				{
					Address:   strings.ToLower(UNIETH),
					Symbol:    "uniETH",
					Decimals:  18,
					Name:      "uniETH",
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
		paused         bool
		totalSupply    *big.Int
		currentReserve *big.Int
	)

	getPoolStateRequest := u.ethrpcClient.NewRequest().SetContext(ctx)
	getPoolStateRequest.AddCall(&ethrpc.Call{
		ABI:    StakingABI,
		Target: Staking,
		Method: StakingMethodPaused,
		Params: []interface{}{},
	}, []interface{}{&paused})
	getPoolStateRequest.AddCall(&ethrpc.Call{
		ABI:    StakingABI,
		Target: Staking,
		Method: StakingMethodCurrentReserve,
		Params: []interface{}{},
	}, []interface{}{&currentReserve})
	getPoolStateRequest.AddCall(&ethrpc.Call{
		ABI:    RockXETHABI,
		Target: UNIETH,
		Method: UniETHMethodTotalSupply,
		Params: []interface{}{},
	}, []interface{}{&totalSupply})

	resp, err := getPoolStateRequest.TryAggregate()
	if err != nil {
		return PoolExtra{}, 0, err
	}
	if resp.BlockNumber == nil {
		resp.BlockNumber = big.NewInt(0)
	}

	return PoolExtra{
		Paused:         paused,
		TotalSupply:    totalSupply,
		CurrentReserve: currentReserve,
	}, resp.BlockNumber.Uint64(), nil
}
