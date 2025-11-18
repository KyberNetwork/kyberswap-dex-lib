package gsm4626

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/erc4626"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
)

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

type PoolTracker struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool,
	_ poolpkg.GetNewPoolStateParams) (entity.Pool, error) {
	logger.Infof("start get new state %v", p.Address)
	defer func() {
		logger.Infof("finish get new pool state %v", p.Address)
	}()

	var (
		canSwap         bool
		currentExposure *big.Int
		exposureCap     *big.Int
		rate            *big.Int
		feeStrategy     common.Address
		tokenBalance    *big.Int
	)
	resp, err := t.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    gsm4626ABI,
			Target: p.Address,
			Method: gsmMethodCanSwap,
		}, []any{&canSwap}).
		AddCall(&ethrpc.Call{
			ABI:    gsm4626ABI,
			Target: p.Address,
			Method: gsmMethodGetAvailableLiquidity,
		}, []any{&currentExposure}).
		AddCall(&ethrpc.Call{
			ABI:    gsm4626ABI,
			Target: p.Address,
			Method: gsmMethodGetExposureCap,
		}, []any{&exposureCap}).
		AddCall(&ethrpc.Call{
			ABI:    erc4626.ABI,
			Target: p.Tokens[1].Address,
			Method: erc4626.ERC4626MethodConvertToAssets,
			Params: []any{ray.ToBig()},
		}, []any{&rate}). // convertToAssets(amt) = amt * rate() / ray
		AddCall(&ethrpc.Call{
			ABI:    gsm4626ABI,
			Target: p.Address,
			Method: gsmMethodGetFeeStrategy,
		}, []any{&feeStrategy}).
		AddCall(&ethrpc.Call{
			ABI:    abi.Erc20ABI,
			Target: p.Tokens[0].Address,
			Method: abi.Erc20BalanceOfMethod,
			Params: []any{common.HexToAddress(p.Address)},
		}, []any{&tokenBalance}).
		Aggregate()
	if err != nil {
		return p, err
	}

	var (
		sellFee *big.Int
		buyFee  *big.Int
	)
	if !eth.IsZeroAddress(feeStrategy) {
		if _, err = t.ethrpcClient.NewRequest().SetContext(ctx).
			SetBlockNumber(resp.BlockNumber).
			AddCall(&ethrpc.Call{
				ABI:    feeStrategyABI,
				Target: feeStrategy.String(),
				Method: feeStrategyMethodGetSellFee,
				Params: []any{percentageFactor.ToBig()},
			}, []any{&sellFee}).
			AddCall(&ethrpc.Call{
				ABI:    feeStrategyABI,
				Target: feeStrategy.String(),
				Method: feeStrategyMethodGetBuyFee,
				Params: []any{percentageFactor.ToBig()},
			}, []any{&buyFee}).
			Aggregate(); err != nil {
			return p, nil
		}
	} else {
		sellFee, buyFee = new(big.Int), new(big.Int)
	}

	extraBytes, err := json.Marshal(Extra{
		CanSwap:         canSwap,
		CurrentExposure: uint256.MustFromBig(currentExposure),
		ExposureCap:     uint256.MustFromBig(exposureCap),
		Rate:            uint256.MustFromBig(rate),
		BuyFee:          uint256.MustFromBig(buyFee),
		SellFee:         uint256.MustFromBig(sellFee),
	})
	if err != nil {
		return p, err
	}
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.Reserves = []string{tokenBalance.String(), currentExposure.String()}

	if resp.BlockNumber != nil {
		p.BlockNumber = resp.BlockNumber.Uint64()
	}

	return p, nil
}
