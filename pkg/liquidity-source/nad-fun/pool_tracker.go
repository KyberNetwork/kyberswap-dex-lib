package nadfun

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
)

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	var (
		curveData struct {
			RealMonReserve          *big.Int
			RealTokenReserve        *big.Int
			VirtualMonReserve       *big.Int
			VirtualTokenReserve     *big.Int
			K                       *big.Int
			TargetTokenAmount       *big.Int
			InitVirtualMonReserve   *big.Int
			InitVirtualTokenReserve *big.Int
		}
		feeConfigData struct {
			DeployFeeAmount   *big.Int
			GraduateFeeAmount *big.Int
			ProtocolFee       *big.Int
		}
		isLocked    bool
		isGraduated bool
	)

	tokenAddress := common.HexToAddress(p.Tokens[1].Address)

	req := t.ethrpcClient.NewRequest().SetContext(ctx)

	req.AddCall(&ethrpc.Call{
		ABI:    bondingCurveABI,
		Target: t.config.BondingCurveAddress,
		Method: "curves",
		Params: []any{tokenAddress},
	}, []any{&curveData})

	req.AddCall(&ethrpc.Call{
		ABI:    bondingCurveABI,
		Target: t.config.BondingCurveAddress,
		Method: "feeConfig",
		Params: nil,
	}, []any{&feeConfigData})

	req.AddCall(&ethrpc.Call{
		ABI:    bondingCurveABI,
		Target: t.config.BondingCurveAddress,
		Method: "isLocked",
		Params: []any{tokenAddress},
	}, []any{&isLocked})

	req.AddCall(&ethrpc.Call{
		ABI:    bondingCurveABI,
		Target: t.config.BondingCurveAddress,
		Method: "isGraduated",
		Params: []any{tokenAddress},
	}, []any{&isGraduated})

	res, err := req.Aggregate()
	if err != nil {
		return entity.Pool{}, err
	}

	extra := Extra{
		IsLocked:      isLocked,
		IsGraduated:   isGraduated,
		VirtualNative: uint256.MustFromBig(curveData.VirtualMonReserve),
		VirtualToken:  uint256.MustFromBig(curveData.VirtualTokenReserve),
		K:             uint256.MustFromBig(curveData.K),
		TargetToken:   uint256.MustFromBig(curveData.TargetTokenAmount),
		ProtocolFee:   uint256.MustFromBig(feeConfigData.ProtocolFee),
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves{
		curveData.RealMonReserve.String(),
		curveData.RealTokenReserve.String(),
	}
	p.BlockNumber = res.BlockNumber.Uint64()
	p.Timestamp = time.Now().Unix()

	return p, nil
}
