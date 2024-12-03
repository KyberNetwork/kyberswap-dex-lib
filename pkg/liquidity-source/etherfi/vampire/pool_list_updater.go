package etherfivampire

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/etherfi/common"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
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
			Exchange:  string(valueobject.ExchangeEtherfiVampire),
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{unlimitedReserve, unlimitedReserve, unlimitedReserve, unlimitedReserve},
			Tokens: []*entity.PoolToken{
				{Address: common.STETH, Symbol: "stETH", Decimals: 18, Swappable: true},
				{Address: common.WSTETH, Symbol: "wstETH", Decimals: 18, Swappable: true},
				{Address: common.EETH, Symbol: "eETH", Decimals: 18, Swappable: true},
				{Address: common.WEETH, Symbol: "weETH", Decimals: 18, Swappable: true},
			},
			BlockNumber: blockNumber,
			Extra:       string(extraBytes),
		},
	}, nil, nil
}

func getPoolExtra(
	ctx context.Context,
	ethrpcClient *ethrpc.Client,
	overrides map[gethcommon.Address]gethclient.OverrideAccount,
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
		Target: common.STETH,
		Method: "getTotalPooledEther",
		Params: []interface{}{},
	}, []interface{}{&poolExtra.StETH.TotalPooledEther})

	r.AddCall(&ethrpc.Call{
		ABI:    stETHABI,
		Target: common.STETH,
		Method: "getTotalShares",
		Params: []interface{}{},
	}, []interface{}{&poolExtra.StETH.TotalShares})

	// poolExtra.StETHTokenInfo
	r.AddCall(&ethrpc.Call{
		ABI:    vampireABI,
		Target: vampire,
		Method: "tokenInfos",
		Params: []interface{}{gethcommon.HexToAddress(common.STETH)},
	}, []interface{}{&tokenInfo})

	r.AddCall(&ethrpc.Call{
		ABI:    vampireABI,
		Target: vampire,
		Method: "timeBoundCapRefreshInterval",
		Params: []interface{}{},
	}, []interface{}{&poolExtra.Vampire.TimeBoundCapRefreshInterval})

	r.AddCall(&ethrpc.Call{
		ABI:    vampireABI,
		Target: vampire,
		Method: "quoteStEthWithCurve",
		Params: []interface{}{},
	}, []interface{}{&poolExtra.Vampire.QuoteStEthWithCurve})

	// poolExtra.LiquidityPool
	r.AddCall(&ethrpc.Call{
		ABI:    liquidityPoolABI,
		Target: common.LiquidityPool,
		Method: "getTotalPooledEther",
		Params: []interface{}{},
	}, []interface{}{&poolExtra.LiquidityPool.TotalPooledEther})

	// poolExtra.EETH
	r.AddCall(&ethrpc.Call{
		ABI:    eETHABI,
		Target: common.EETH,
		Method: "totalShares",
		Params: []interface{}{},
	}, []interface{}{&poolExtra.EETH.TotalShares})

	// Call RPC
	resp, err := r.TryAggregate()
	if err != nil {
		return PoolExtra{}, 0, err
	}
	if resp.BlockNumber == nil {
		resp.BlockNumber = bignumber.ZeroBI
	}

	// Update poolExtra.StETHTokenInfo from tokenInfo
	poolExtra.StETHTokenInfo.DiscountInBasisPoints = big.NewInt(int64(tokenInfo.DiscountInBasisPoints))
	poolExtra.StETHTokenInfo.TotalDepositedThisPeriod = tokenInfo.TotalDepositedThisPeriod
	poolExtra.StETHTokenInfo.TotalDeposited = tokenInfo.TotalDeposited
	poolExtra.StETHTokenInfo.TimeBoundCapClockStartTime = tokenInfo.TimeBoundCapClockStartTime
	poolExtra.StETHTokenInfo.TimeBoundCapInEther = tokenInfo.TimeBoundCapInEther
	poolExtra.StETHTokenInfo.TotalCapInEther = tokenInfo.TotalCapInEther

	// Get and update CurvePoolInfo
	curvePoolInfo, err := getCurvePoolInfo(ctx, ethrpcClient, overrides)
	if err != nil {
		return PoolExtra{}, 0, err
	}
	poolExtra.CurveStETHToETH = curvePoolInfo

	return poolExtra, resp.BlockNumber.Uint64(), nil
}

func getCurvePoolInfo(
	ctx context.Context,
	ethrpcClient *ethrpc.Client,
	overrides map[gethcommon.Address]gethclient.OverrideAccount,
) (CurvePoolInfo, error) {
	var (
		curvePlainExtra CurvePlainExtra
		curvePoolInfo   CurvePoolInfo
	)

	r := ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		r.SetOverrides(overrides)
	}

	r.AddCall(&ethrpc.Call{
		ABI:    curvePlainABI,
		Target: curveStETHToETHPool,
		Method: "initial_A",
		Params: nil,
	}, []interface{}{&curvePlainExtra.InitialA})

	r.AddCall(&ethrpc.Call{
		ABI:    curvePlainABI,
		Target: curveStETHToETHPool,
		Method: "future_A",
		Params: nil,
	}, []interface{}{&curvePlainExtra.FutureA})

	r.AddCall(&ethrpc.Call{
		ABI:    curvePlainABI,
		Target: curveStETHToETHPool,
		Method: "initial_A_time",
		Params: nil,
	}, []interface{}{&curvePlainExtra.InitialATime})

	r.AddCall(&ethrpc.Call{
		ABI:    curvePlainABI,
		Target: curveStETHToETHPool,
		Method: "future_A_time",
		Params: nil,
	}, []interface{}{&curvePlainExtra.FutureATime})

	r.AddCall(&ethrpc.Call{
		ABI:    curvePlainABI,
		Target: curveStETHToETHPool,
		Method: "fee",
		Params: nil,
	}, []interface{}{&curvePlainExtra.SwapFee})

	r.AddCall(&ethrpc.Call{
		ABI:    curvePlainABI,
		Target: curveStETHToETHPool,
		Method: "admin_fee",
		Params: nil,
	}, []interface{}{&curvePlainExtra.AdminFee})

	nCoins := 2
	balances := make([]*big.Int, nCoins)

	for i := 0; i < nCoins; i++ {
		r.AddCall(&ethrpc.Call{
			ABI:    curvePlainABI,
			Target: curveStETHToETHPool,
			Method: "balances",
			Params: []interface{}{big.NewInt(int64(i))},
		}, []interface{}{&balances[i]})
	}

	if _, err := r.TryAggregate(); err != nil {
		return CurvePoolInfo{}, err
	}

	curvePoolInfo.Reserves = make([]string, nCoins+1)
	for i := 0; i < nCoins; i++ {
		curvePoolInfo.Reserves[i] = balances[i].String()
	}
	// The last reserve is from the balanceV1 pool,
	// we don't need to use it so set it to "0" instead of tracking.
	curvePoolInfo.Reserves[nCoins] = "0"

	extraBytes, err := json.Marshal(curvePlainExtra)
	if err != nil {
		return CurvePoolInfo{}, err
	}
	curvePoolInfo.Extra = string(extraBytes)

	// Since staticExtra doesn't change, we can hardcode it here.
	curvePoolInfo.StaticExtra = "{\"APrecision\":\"100\",\"LpToken\":\"0x06325440D014e39736583c165C2963BA99fAf14E\",\"IsNativeCoin\":[true,false]}"

	return curvePoolInfo, nil
}
