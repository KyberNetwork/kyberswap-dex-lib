package reth

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"

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

	extra, blockNumber, err := getExtra(ctx, u.ethrpcClient)
	if err != nil {
		return nil, nil, err
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return nil, nil, err
	}

	return []entity.Pool{
		{
			Address:   strings.ToLower(RocketDepositPool),
			Exchange:  string(valueobject.ExchangeRocketPoolRETH),
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{reserves, reserves},
			Tokens: []*entity.PoolToken{
				{Address: strings.ToLower(WETH), Symbol: "WETH", Decimals: 18, Name: "Wrapped Ether", Swappable: true},
				{Address: strings.ToLower(RocketTokenRETH), Symbol: "rETH", Decimals: 18, Name: "Rocket Pool ETH", Swappable: true},
			},
			BlockNumber: blockNumber,
			Extra:       string(extraBytes),
		},
	}, nil, nil
}

func getExtra(ctx context.Context, ethrpcClient *ethrpc.Client) (PoolExtra, uint64, error) {
	var poolExtra PoolExtra
	balanceAt, err := ethrpcClient.BalanceAt(ctx, common.HexToAddress(RocketTokenRETH), nil)
	if err != nil {
		return poolExtra, 0, err
	}
	poolExtra.RETHBalance = balanceAt

	rpcCalls := ethrpcClient.NewRequest().SetContext(ctx)

	rpcCalls.AddCall(&ethrpc.Call{
		ABI:    RocketDAOProtocolSettingsDepositABI,
		Target: RocketDAOProtocolSettingsDeposit,
		Method: "getDepositEnabled",
		Params: []interface{}{},
	}, []interface{}{&poolExtra.DepositEnabled})

	rpcCalls.AddCall(&ethrpc.Call{
		ABI:    RocketDAOProtocolSettingsDepositABI,
		Target: RocketDAOProtocolSettingsDeposit,
		Method: "getMinimumDeposit",
		Params: []interface{}{},
	}, []interface{}{&poolExtra.MinimumDeposit})

	rpcCalls.AddCall(&ethrpc.Call{
		ABI:    RocketDAOProtocolSettingsDepositABI,
		Target: RocketDAOProtocolSettingsDeposit,
		Method: "getMaximumDepositPoolSize",
		Params: []interface{}{},
	}, []interface{}{&poolExtra.MaximumDepositPoolSize})

	rpcCalls.AddCall(&ethrpc.Call{
		ABI:    RocketDAOProtocolSettingsDepositABI,
		Target: RocketDAOProtocolSettingsDeposit,
		Method: "getAssignDepositsEnabled",
		Params: []interface{}{},
	}, []interface{}{&poolExtra.AssignDepositsEnabled})

	rpcCalls.AddCall(&ethrpc.Call{
		ABI:    RocketDAOProtocolSettingsDepositABI,
		Target: RocketDAOProtocolSettingsDeposit,
		Method: "getDepositFee",
		Params: []interface{}{},
	}, []interface{}{&poolExtra.DepositFee})

	rpcCalls.AddCall(&ethrpc.Call{
		ABI:    RocketVaultABI,
		Target: RocketVault,
		Method: "balanceOf",
		Params: []interface{}{RocketDepositPool},
	}, []interface{}{&poolExtra.Balance})

	rpcCalls.AddCall(&ethrpc.Call{
		ABI:    RocketMinipoolQueueABI,
		Target: RocketMinipoolQueue,
		Method: "getEffectiveCapacity",
		Params: []interface{}{},
	}, []interface{}{&poolExtra.EffectiveCapacity})

	rpcCalls.AddCall(&ethrpc.Call{
		ABI:    RocketNetworkBalancesABI,
		Target: RocketNetworkBalances,
		Method: "getTotalETHBalance",
		Params: []interface{}{},
	}, []interface{}{&poolExtra.TotalETHBalance})

	rpcCalls.AddCall(&ethrpc.Call{
		ABI:    RocketNetworkBalancesABI,
		Target: RocketNetworkBalances,
		Method: "getTotalRETHSupply",
		Params: []interface{}{},
	}, []interface{}{&poolExtra.TotalRETHSupply})

	rpcCalls.AddCall(&ethrpc.Call{
		ABI:    RocketDepositPoolABI,
		Target: RocketDepositPool,
		Method: "getExcessBalance",
		Params: []interface{}{},
	}, []interface{}{&poolExtra.ExcessBalance})

	resp, err := rpcCalls.TryAggregate()
	if err != nil {
		return PoolExtra{}, 0, err
	}

	if resp.BlockNumber == nil {
		resp.BlockNumber = big.NewInt(0)
	}

	return poolExtra, resp.BlockNumber.Uint64(), nil
}
