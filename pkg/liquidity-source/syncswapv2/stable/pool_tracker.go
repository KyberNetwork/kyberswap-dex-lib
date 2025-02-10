package syncswapv2stable

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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/syncswapv2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *syncswapv2.Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexTypeSyncSwapV2Stable, NewPoolTracker)

func NewPoolTracker(
	config *syncswapv2.Config,
	ethrpcClient *ethrpc.Client,
) *PoolTracker {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("[%s] Start getting new state of pool", p.Type)

	var (
		swapFee0To1, swapFee1To0                             *big.Int
		token0PrecisionMultiplier, token1PrecisionMultiplier *big.Int
		vaultAddress                                         common.Address
		reserves                                             = make([]*big.Int, len(p.Tokens))
		A                                                    uint64
	)

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)

	calls.AddCall(&ethrpc.Call{
		ABI:    stablePoolABI,
		Target: p.Address,
		Method: poolMethodGetSwapFee,
		Params: []interface{}{
			common.HexToAddress(addressZero),
			common.HexToAddress(p.Tokens[0].Address),
			common.HexToAddress(p.Tokens[1].Address),
			[]byte{},
		},
	}, []interface{}{&swapFee0To1})

	calls.AddCall(&ethrpc.Call{
		ABI:    stablePoolABI,
		Target: p.Address,
		Method: poolMethodGetSwapFee,
		Params: []interface{}{
			common.HexToAddress(addressZero),
			common.HexToAddress(p.Tokens[1].Address),
			common.HexToAddress(p.Tokens[0].Address),
			[]byte{},
		},
	}, []interface{}{&swapFee1To0})

	calls.AddCall(&ethrpc.Call{
		ABI:    stablePoolABI,
		Target: p.Address,
		Method: poolMethodGetReserves,
		Params: nil,
	}, []interface{}{&reserves})

	calls.AddCall(&ethrpc.Call{
		ABI:    stablePoolABI,
		Target: p.Address,
		Method: poolMethodToken0PrecisionMultiplier,
		Params: nil,
	}, []interface{}{&token0PrecisionMultiplier})

	calls.AddCall(&ethrpc.Call{
		ABI:    stablePoolABI,
		Target: p.Address,
		Method: poolMethodToken1PrecisionMultiplier,
		Params: nil,
	}, []interface{}{&token1PrecisionMultiplier})

	calls.AddCall(&ethrpc.Call{
		ABI:    stablePoolABI,
		Target: p.Address,
		Method: poolMethodVault,
		Params: nil,
	}, []interface{}{&vaultAddress})

	calls.AddCall(&ethrpc.Call{
		ABI:    stablePoolABI,
		Target: p.Address,
		Method: poolMethodGetA,
		Params: nil,
	}, []interface{}{&A})

	if _, err := calls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"address": p.Address,
			"error":   err,
		}).Errorf("failed to get state of the pool")
		return entity.Pool{}, err
	}

	extraBytes, err := json.Marshal(ExtraStablePool{
		SwapFee0To1:               uint256.MustFromBig(swapFee0To1),
		SwapFee1To0:               uint256.MustFromBig(swapFee1To0),
		Token0PrecisionMultiplier: uint256.MustFromBig(token0PrecisionMultiplier),
		Token1PrecisionMultiplier: uint256.MustFromBig(token1PrecisionMultiplier),
		VaultAddress:              vaultAddress.Hex(),
		A:                         uint256.NewInt(A),
	})
	if err != nil {
		logger.WithFields(logger.Fields{
			"address": p.Address,
			"error":   err,
		}).Errorf("failed to marshal extra data")

		return entity.Pool{}, err
	}

	p.Reserves = entity.PoolReserves{reserves[0].String(), reserves[1].String()}
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("[%s] Finish getting new state of pool", p.Type)

	return p, nil
}
