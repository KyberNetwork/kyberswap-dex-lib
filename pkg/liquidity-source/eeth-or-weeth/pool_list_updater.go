package eethorweeth

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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolListUpdater struct {
	ethrpcClient   *ethrpc.Client
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

	extra, blockNumber, err := getPoolExtra(ctx, u.ethrpcClient, nil)
	if err != nil {
		return nil, nil, err
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return nil, nil, err
	}

	return []entity.Pool{
		{
			Address:   strings.ToLower(vampire),
			Exchange:  string(valueobject.ExchangeEETHOrWEETH),
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{},
			Tokens: []*entity.PoolToken{
				{Address: stETH, Symbol: "stETH", Decimals: 18, Swappable: true},
				{Address: wstETH, Symbol: "wstETH", Decimals: 18, Swappable: true},
				{Address: eETH, Symbol: "eETH", Decimals: 18, Swappable: true},
				{Address: weETH, Symbol: "weETH", Decimals: 18, Swappable: true},
			},
			BlockNumber: blockNumber,
			Extra:       string(extraBytes),
		},
	}, nil, nil
}

func getPoolExtra(
	ctx context.Context,
	ethrpcClient *ethrpc.Client,
	overrides map[common.Address]gethclient.OverrideAccount,
) (PoolExtra, uint64, error) {
	var (
		poolExtra PoolExtra
		tokenInfo VampireTokenInfo
	)

	r := ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		r.SetOverrides(overrides)
	}

	// poolExtra.StETH
	r.AddCall(&ethrpc.Call{
		ABI:    stETHABI,
		Target: stETH,
		Method: "getTotalPooledEther",
		Params: []interface{}{},
	}, []interface{}{&poolExtra.StETH.TotalPooledEther})

	r.AddCall(&ethrpc.Call{
		ABI:    stETHABI,
		Target: stETH,
		Method: "getTotalShares",
		Params: []interface{}{},
	}, []interface{}{&poolExtra.StETH.TotalShares})

	// poolExtra.StETHTokenInfo
	r.AddCall(&ethrpc.Call{
		ABI:    vampireABI,
		Target: vampire,
		Method: "tokenInfos",
		Params: []interface{}{common.HexToAddress(stETH)},
	}, []interface{}{&tokenInfo})

	r.AddCall(&ethrpc.Call{
		ABI:    vampireABI,
		Target: vampire,
		Method: "timeBoundCapRefreshInterval",
		Params: []interface{}{},
	}, []interface{}{&poolExtra.TimeBoundCapRefreshInterval})

	// poolExtra.EtherFiPool
	r.AddCall(&ethrpc.Call{
		ABI:    liquidityPoolABI,
		Target: liquidityPool,
		Method: "getTotalPooledEther",
		Params: []interface{}{},
	}, []interface{}{&poolExtra.EtherFiPool.TotalPooledEther})

	// poolExtra.EETH
	r.AddCall(&ethrpc.Call{
		ABI:    eETHABI,
		Target: eETH,
		Method: "totalShares",
		Params: []interface{}{},
	}, []interface{}{&poolExtra.EETH.TotalShares})

	// Call RPC
	resp, err := r.TryAggregate()
	if err != nil {
		return PoolExtra{}, 0, err
	}
	if resp.BlockNumber == nil {
		resp.BlockNumber = big.NewInt(0)
	}

	// Update poolExtra.StETHTokenInfo from tokenInfo
	poolExtra.StETHTokenInfo.DiscountInBasisPoints = tokenInfo.DiscountInBasisPoints
	poolExtra.StETHTokenInfo.TotalDepositedThisPeriod = tokenInfo.TotalDepositedThisPeriod
	poolExtra.StETHTokenInfo.TotalDeposited = tokenInfo.TotalDeposited
	poolExtra.StETHTokenInfo.TimeBoundCapClockStartTime = tokenInfo.TimeBoundCapClockStartTime
	poolExtra.StETHTokenInfo.TimeBoundCapInEther = tokenInfo.TimeBoundCapInEther
	poolExtra.StETHTokenInfo.TotalCapInEther = tokenInfo.TotalCapInEther

	return poolExtra, resp.BlockNumber.Uint64(), nil
}
