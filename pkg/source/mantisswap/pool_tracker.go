package mantisswap

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/bytedance/sonic"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
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
	}).Infof("[%s] Start getting new states of pool", p.Type)

	var extra Extra
	if err := sonic.Unmarshal([]byte(p.Extra), &extra); err != nil {
		logger.Errorf("failed to unmarshal extra with err %v", err)
		return entity.Pool{}, err
	}

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)
	calls.AddCall(&ethrpc.Call{
		ABI:    MainPoolABI,
		Target: d.config.MainPoolAddress,
		Method: mainPoolMethodPaused,
		Params: nil,
	}, []interface{}{&extra.Paused})
	calls.AddCall(&ethrpc.Call{
		ABI:    MainPoolABI,
		Target: d.config.MainPoolAddress,
		Method: mainPoolMethodSwapAlloed,
		Params: nil,
	}, []interface{}{&extra.SwapAllowed})
	calls.AddCall(&ethrpc.Call{
		ABI:    MainPoolABI,
		Target: d.config.MainPoolAddress,
		Method: mainPoolMethodBaseFee,
		Params: nil,
	}, []interface{}{&extra.BaseFee})
	calls.AddCall(&ethrpc.Call{
		ABI:    MainPoolABI,
		Target: d.config.MainPoolAddress,
		Method: mainPoolMethodLpRatio,
		Params: nil,
	}, []interface{}{&extra.LpRatio})
	calls.AddCall(&ethrpc.Call{
		ABI:    MainPoolABI,
		Target: d.config.MainPoolAddress,
		Method: mainPoolMethodSlippageA,
		Params: nil,
	}, []interface{}{&extra.SlippageA})
	calls.AddCall(&ethrpc.Call{
		ABI:    MainPoolABI,
		Target: d.config.MainPoolAddress,
		Method: mainPoolMethodSlippageK,
		Params: nil,
	}, []interface{}{&extra.SlippageK})
	calls.AddCall(&ethrpc.Call{
		ABI:    MainPoolABI,
		Target: d.config.MainPoolAddress,
		Method: mainPoolMethodSlippageN,
		Params: nil,
	}, []interface{}{&extra.SlippageN})
	for _, token := range p.Tokens {
		lp := extra.LPs[token.Address]
		calls.AddCall(&ethrpc.Call{
			ABI:    LPABI,
			Target: lp.Address,
			Method: lpMethodDecimals,
			Params: nil,
		}, []interface{}{&lp.Decimals})
		calls.AddCall(&ethrpc.Call{
			ABI:    LPABI,
			Target: lp.Address,
			Method: lpMethodAsset,
			Params: nil,
		}, []interface{}{&lp.Asset})
		calls.AddCall(&ethrpc.Call{
			ABI:    LPABI,
			Target: lp.Address,
			Method: lpMethodLiability,
			Params: nil,
		}, []interface{}{&lp.Liability})
		calls.AddCall(&ethrpc.Call{
			ABI:    LPABI,
			Target: lp.Address,
			Method: lpMethodLiabilityLimit,
			Params: nil,
		}, []interface{}{&lp.LiabilityLimit})
	}
	if _, err := calls.Aggregate(); err != nil {
		logger.Errorf("failed to aggregate calls with err %v", err)
		return entity.Pool{}, err
	}

	reserves := make([]string, len(p.Tokens))
	for i, token := range p.Tokens {
		reserves[i] = extra.LPs[token.Address].Asset.String()
	}

	extraBytes, err := sonic.Marshal(extra)
	if err != nil {
		logger.Errorf("failed to marshal extra with err %v", err)
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = reserves
	p.Timestamp = time.Now().Unix()

	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("[%s] Finish getting new state of pool", p.Type)

	return p, nil
}
