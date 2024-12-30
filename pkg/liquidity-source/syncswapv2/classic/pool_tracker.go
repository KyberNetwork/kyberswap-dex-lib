package syncswapv2classic

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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/syncswap"
)

type PoolTracker struct {
	config       *syncswap.Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(
	config *syncswap.Config,
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
		swapFee0To1, swapFee1To0 *big.Int
		reserves                 = make([]*big.Int, len(p.Tokens))
		vaultAddress             common.Address
	)

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)

	calls.AddCall(&ethrpc.Call{
		ABI:    classicPoolABI,
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
		ABI:    classicPoolABI,
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
		ABI:    classicPoolABI,
		Target: p.Address,
		Method: poolMethodGetReserves,
		Params: nil,
	}, []interface{}{&reserves})

	calls.AddCall(&ethrpc.Call{
		ABI:    classicPoolABI,
		Target: p.Address,
		Method: poolMethodVault,
		Params: nil,
	}, []interface{}{&vaultAddress})

	if _, err := calls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"address": p.Address,
			"error":   err,
		}).Errorf("failed to get state of the pool")
		return entity.Pool{}, err
	}

	extraBytes, err := json.Marshal(ExtraClassicPool{
		SwapFee0To1:  uint256.MustFromBig(swapFee0To1),
		SwapFee1To0:  uint256.MustFromBig(swapFee1To0),
		VaultAddress: vaultAddress.Hex(),
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
