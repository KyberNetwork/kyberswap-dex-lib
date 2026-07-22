package machima

import (
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/abis"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
)

// PoolFactory decodes PoolCreated logs so the ticks-based worker can pick up pools created after
// the initial subgraph backfill.
//
// Machima's factory is an unmodified UniV3 factory, so the UniV3 ABI is reused rather than
// duplicated: scanning the deployed factory for the standard PoolCreated topic
// (0x783cca1c...) returns real logs.
type PoolFactory struct {
	config              *Config
	poolCreatedEventIDs map[common.Hash]struct{}
}

var _ = poolfactory.RegisterFactoryC(DexType, NewPoolFactory)

func NewPoolFactory(config *Config) *PoolFactory {
	return &PoolFactory{
		config: config,
		poolCreatedEventIDs: map[common.Hash]struct{}{
			abis.UniswapV3FactoryABI.Events["PoolCreated"].ID: {},
		},
	}
}

func (f *PoolFactory) IsEventSupported(event common.Hash) bool {
	_, ok := f.poolCreatedEventIDs[event]
	return ok
}

func (f *PoolFactory) DecodePoolCreated(event ethtypes.Log) (*entity.Pool, error) {
	created, err := abis.UniswapV3FactoryFilterer.ParsePoolCreated(event)
	if err != nil {
		return nil, err
	}

	token0 := hexutil.Encode(created.Token0[:])
	token1 := hexutil.Encode(created.Token1[:])

	// PoolCreated does not say which side is the launched token, so derive it the same way the
	// router's _classifyPair does: the traded token is the side that is not a counter asset, and
	// when both sides are counter assets it is whichever one is XMA.
	tradedToken, ok := f.resolveTradedToken(token0, token1)
	if !ok {
		return nil, ErrInvalidPair
	}

	staticExtra, err := json.Marshal(StaticExtra{
		Token:         tradedToken,
		RouterAddress: strings.ToLower(f.config.RouterAddress),
		WETH:          strings.ToLower(f.config.WETH),
		USDC:          strings.ToLower(f.config.USDC),
		XMA:           strings.ToLower(f.config.XMA),
	})
	if err != nil {
		return nil, err
	}

	// tickSpacing comes straight off the event, so the tracker's first tick sweep needs no extra
	// RPC to learn it.
	extra, err := json.Marshal(Extra{Extra: uniswapv3.Extra{TickSpacing: created.TickSpacing.Uint64()}})
	if err != nil {
		return nil, err
	}

	swapFee, _ := created.Fee.Float64()

	return &entity.Pool{
		Address:   hexutil.Encode(created.Pool[:]),
		Exchange:  f.config.DexID,
		Type:      DexType,
		SwapFee:   swapFee,
		Timestamp: time.Now().Unix(),
		// Placeholders; the tracker fills real reserves on the first refresh.
		Reserves: entity.PoolReserves{"0", "0"},
		Tokens: []*entity.PoolToken{
			{Address: token0, Swappable: true},
			{Address: token1, Swappable: true},
		},
		Extra:       string(extra),
		StaticExtra: string(staticExtra),
		BlockNumber: event.BlockNumber,
	}, nil
}

func (f *PoolFactory) resolveTradedToken(token0, token1 string) (string, bool) {
	xma := strings.ToLower(f.config.XMA)
	is0Counter, is1Counter := f.isCounterAsset(token0), f.isCounterAsset(token1)

	switch {
	case is0Counter && !is1Counter:
		return token1, true
	case is1Counter && !is0Counter:
		return token0, true
	case is0Counter && is1Counter:
		if token0 == xma && token1 != xma {
			return token0, true
		}
		if token1 == xma && token0 != xma {
			return token1, true
		}
	}
	return "", false
}

func (f *PoolFactory) isCounterAsset(token string) bool {
	return token == strings.ToLower(f.config.WETH) ||
		token == strings.ToLower(f.config.USDC) ||
		token == strings.ToLower(f.config.XMA)
}
