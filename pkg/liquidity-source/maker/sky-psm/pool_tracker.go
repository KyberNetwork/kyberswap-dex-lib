package skypsm

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	sky "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/maker/savingsdai"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

type PoolTracker struct {
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryE0(DexType, NewPoolTracker)

func NewPoolTracker(ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{ethrpcClient: ethrpcClient}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, pool.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {

	logger.WithFields(logger.Fields{
		"exchange": p.Exchange,
		"address":  p.Address,
	}).Infof("[%s] Start getting new state of pool", p.Type)

	staticExtra := StaticExtra{}
	err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra)
	if err != nil {
		return entity.Pool{}, err
	}

	blockTimestamp := uint64(time.Now().Unix()) + sky.Blocktime

	var rate *big.Int
	balances := make([]*big.Int, len(p.Tokens))
	calls := t.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		calls.SetOverrides(overrides)
	}
	calls.AddCall(&ethrpc.Call{
		ABI:    ssrOracleABI,
		Target: staticExtra.RateProvider,
		Method: ssrOracleMethodGetConversionRate,
		Params: []any{new(big.Int).SetUint64(blockTimestamp)},
	}, []any{&rate})
	for i, token := range p.Tokens {
		calls.AddCall(&ethrpc.Call{
			ABI:    abi.Erc20ABI,
			Target: token.Address,
			Method: abi.Erc20BalanceOfMethod,
			Params: []any{staticExtra.Pocket},
		}, []any{&balances[i]})
	}
	_, err = calls.Aggregate()
	if err != nil {
		return p, err
	}

	extraBytes, err := json.Marshal(Extra{
		Rate:           uint256.MustFromBig(rate),
		BlockTimestamp: blockTimestamp,
	})
	if err != nil {
		return p, err
	}
	p.Extra = string(extraBytes)
	for i, balance := range balances {
		p.Reserves[i] = balance.String()
	}
	p.Timestamp = time.Now().Unix()

	logger.WithFields(logger.Fields{
		"exchange": p.Exchange,
		"address":  p.Address,
	}).Infof("[%s] Finish getting new state of pool", p.Type)

	return p, nil
}
