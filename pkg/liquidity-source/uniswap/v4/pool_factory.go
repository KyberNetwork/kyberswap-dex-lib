package uniswapv4

import (
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	abis "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var _ = poolfactory.RegisterFactoryC(DexType, NewPoolFactory)

type PoolFactory struct {
	cfg                 *Config
	poolCreatedEventIds map[common.Hash]struct{}
}

func NewPoolFactory(config *Config) *PoolFactory {
	return &PoolFactory{
		cfg: config,
		poolCreatedEventIds: map[common.Hash]struct{}{
			poolManagerABI.Events["Initialize"].ID: {},
		},
	}
}

func (f *PoolFactory) DecodePoolCreated(event ethtypes.Log) (*entity.Pool, error) {
	p, err := poolManagerFilterer.ParseInitialize(event)
	if err != nil {
		return nil, err
	}

	return f.newPool(p, event.BlockNumber)
}

func (f *PoolFactory) IsEventSupported(event common.Hash) bool {
	_, ok := f.poolCreatedEventIds[event]
	return ok
}

func DecodePoolAddress(log ethtypes.Log) (string, error) {
	if len(log.Topics) == 0 || eth.IsZeroAddress(log.Address) {
		return "", nil
	}

	switch log.Topics[0] {
	case poolManagerABI.Events["Initialize"].ID,
		poolManagerABI.Events["Donate"].ID,
		poolManagerABI.Events["ModifyLiquidity"].ID,
		poolManagerABI.Events["ProtocolFeeUpdated"].ID,
		poolManagerABI.Events["Swap"].ID: // these events have the pool address in topic1
		if len(log.Topics) < 2 {
			break
		}
		return hexutil.Encode(log.Topics[1][:]), nil
	}

	return "", nil
}

func (f *PoolFactory) newPool(p *abis.UniswapV4PoolManagerInitialize, blockNbr uint64) (*entity.Pool, error) {
	chainId := valueobject.ChainID(f.cfg.ChainID)
	hook, _ := GetHook(p.Hooks, &HookParam{
		Cfg: f.cfg,
	})

	poolAddress := hexutil.Encode(p.Id[:])

	swapFee, _ := p.Fee.Float64()
	extraBytes, _ := json.Marshal(Extra{
		Extra: &uniswapv3.Extra{
			SqrtPriceX96: p.SqrtPriceX96,
			TickSpacing:  p.TickSpacing.Uint64(),
			Tick:         p.Tick,
		},
	})
	staticExtra := StaticExtra{
		IsNative:               [2]bool{eth.IsZeroAddress(p.Currency0), eth.IsZeroAddress(p.Currency1)},
		Fee:                    uint32(p.Fee.Uint64()),
		TickSpacing:            int32(p.TickSpacing.Int64()),
		HooksAddress:           p.Hooks,
		UniversalRouterAddress: common.HexToAddress(f.cfg.UniversalRouterAddress),
		Permit2Address:         common.HexToAddress(f.cfg.Permit2Address),
		Multicall3Address:      common.HexToAddress(f.cfg.Multicall3Address),
	}
	staticExtraBytes, _ := json.Marshal(staticExtra)

	return &entity.Pool{
		Address:   poolAddress,
		SwapFee:   swapFee,
		Type:      DexType,
		Exchange:  hook.GetExchange(),
		Timestamp: time.Now().Unix(),
		Reserves:  entity.PoolReserves{"0", "0"},
		Tokens: []*entity.PoolToken{
			{Address: currencyToToken(p.Currency0, chainId), Swappable: true},
			{Address: currencyToToken(p.Currency1, chainId), Swappable: true},
		},
		Extra:       string(extraBytes),
		StaticExtra: string(staticExtraBytes),
		BlockNumber: blockNbr,
	}, nil
}

func currencyToToken(currency common.Address, chainId valueobject.ChainID) string {
	if eth.IsZeroAddress(currency) {
		return strings.ToLower(valueobject.WrappedNativeMap[chainId])
	}
	return hexutil.Encode(currency[:])
}
