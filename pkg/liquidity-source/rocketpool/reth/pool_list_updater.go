package reth

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
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

func (u *PoolListUpdater) GetNewPools(ctx context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	if u.hasInitialized {
		return nil, nil, nil
	}

	u.hasInitialized = true

	extra, blockNumber, err := getExtra(ctx, u.ethrpcClient, nil)
	if err != nil {
		return nil, nil, err
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return nil, nil, err
	}

	var tmp big.Int
	reserve0 := tmp.Add(extra.ExcessBalance, extra.RETHBalance).String()
	reserve1 := bignumber.MulDivDown(&tmp, &tmp, extra.TotalRETHSupply, extra.TotalETHBalance).String()

	return []entity.Pool{
		{
			Address:   strings.ToLower(RocketDepositPool),
			Exchange:  valueobject.ExchangeRocketPoolRETH,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{reserve0, reserve1},
			Tokens: []*entity.PoolToken{
				{Address: strings.ToLower(WETH), Symbol: "WETH", Decimals: 18, Swappable: true},
				{Address: strings.ToLower(RocketTokenRETH), Symbol: "rETH", Decimals: 18, Swappable: true},
			},
			BlockNumber: blockNumber,
			Extra:       string(extraBytes),
		},
	}, nil, nil
}

func getExtra(ctx context.Context, ethrpcClient *ethrpc.Client,
	overrides map[common.Address]gethclient.OverrideAccount) (PoolExtra, uint64, error) {
	var poolExtra PoolExtra
	resp, err := ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides).AddCall(&ethrpc.Call{
		ABI:    abi.Multicall3ABI,
		Target: Multicall3,
		Method: abi.Multicall3GetEthBalance,
		Params: []any{common.HexToAddress(RocketTokenRETH)},
	}, []any{&poolExtra.RETHBalance}).AddCall(&ethrpc.Call{
		ABI:    RocketDAOProtocolSettingsDepositABI,
		Target: RocketDAOProtocolSettingsDeposit,
		Method: "getDepositEnabled",
	}, []any{&poolExtra.DepositEnabled}).AddCall(&ethrpc.Call{
		ABI:    RocketDAOProtocolSettingsDepositABI,
		Target: RocketDAOProtocolSettingsDeposit,
		Method: "getMinimumDeposit",
	}, []any{&poolExtra.MinimumDeposit}).AddCall(&ethrpc.Call{
		ABI:    RocketDAOProtocolSettingsDepositABI,
		Target: RocketDAOProtocolSettingsDeposit,
		Method: "getMaximumDepositPoolSize",
	}, []any{&poolExtra.MaximumDepositPoolSize}).AddCall(&ethrpc.Call{
		ABI:    RocketDAOProtocolSettingsDepositABI,
		Target: RocketDAOProtocolSettingsDeposit,
		Method: "getAssignDepositsEnabled",
	}, []any{&poolExtra.AssignDepositsEnabled}).AddCall(&ethrpc.Call{
		ABI:    RocketDAOProtocolSettingsDepositABI,
		Target: RocketDAOProtocolSettingsDeposit,
		Method: "getDepositFee",
	}, []any{&poolExtra.DepositFee}).AddCall(&ethrpc.Call{
		ABI:    RocketVaultABI,
		Target: RocketVault,
		Method: "balanceOf",
		Params: []any{"rocketDepositPool"},
	}, []any{&poolExtra.Balance}).AddCall(&ethrpc.Call{
		ABI:    RocketMinipoolQueueABI,
		Target: RocketMinipoolQueue,
		Method: "getEffectiveCapacity",
	}, []any{&poolExtra.EffectiveCapacity}).AddCall(&ethrpc.Call{
		ABI:    RocketNetworkBalancesABI,
		Target: RocketNetworkBalances,
		Method: "getTotalETHBalance",
	}, []any{&poolExtra.TotalETHBalance}).AddCall(&ethrpc.Call{
		ABI:    RocketNetworkBalancesABI,
		Target: RocketNetworkBalances,
		Method: "getTotalRETHSupply",
	}, []any{&poolExtra.TotalRETHSupply}).AddCall(&ethrpc.Call{
		ABI:    RocketDepositPoolABI,
		Target: RocketDepositPool,
		Method: "getExcessBalance",
	}, []any{&poolExtra.ExcessBalance}).TryAggregate()
	if err != nil {
		return PoolExtra{}, 0, err
	}

	var blockNumber uint64
	if resp.BlockNumber != nil {
		blockNumber = resp.BlockNumber.Uint64()
	}

	return poolExtra, blockNumber, nil
}
