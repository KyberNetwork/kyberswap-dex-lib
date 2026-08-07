package flap

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// tokenCreatedEventTopic is the topic0 of Portal's TokenCreated(uint256,address,uint256,address,
// string,string,string) event - all params non-indexed, per the Portal ABI.
var tokenCreatedEventTopic = portalABI.Events["TokenCreated"].ID

// flapTokenCirculatingSupplyChangedEventTopic is the topic0 of
// FlapTokenCirculatingSupplyChanged(address token, uint256 newSupply), emitted by PortalBase's
// _mintToken/_burnToken at the end of every buy and every sell respectively (see PortalBase.sol). This
// is the anchor event used to trigger a reactive tracker refresh on trades: Portal's trade logs
// (TokenBought/TokenSold) are versioned and more complex to decode, but every trade unconditionally
// touches circulating supply, so this one event topic covers both directions.
var flapTokenCirculatingSupplyChangedEventTopic = portalABI.Events["FlapTokenCirculatingSupplyChanged"].ID

// launchedToDEXEventTopic is the topic0 of LaunchedToDEX(address token, address pool, uint256 amount,
// uint256 eth), emitted once when a token graduates off the bonding curve. Routing it to the pool lets
// the tracker mark the pool permanently disabled (see pool_tracker.go) the block it happens, instead of
// waiting for the next interval/trade-triggered refresh.
var launchedToDEXEventTopic = portalABI.Events["LaunchedToDEX"].ID

// PoolFactory is the live pool-discovery path: it decodes Portal's TokenCreated event so a newly
// launched token appears the block it is created, instead of waiting for the next
// graduatinghot-board bootstrap pass (which is a one-shot backfill only, see pool_list_updater.go -
// a plain board-cursor cannot double as an ongoing "what's new" feed).
type PoolFactory struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = poolfactory.RegisterFactoryCE(DexType, NewPoolFactory)

func NewPoolFactory(config *Config, ethrpcClient *ethrpc.Client) *PoolFactory {
	return &PoolFactory{config: config, ethrpcClient: ethrpcClient}
}

// IsEventSupported reports whether topic0 is Portal's TokenCreated event.
func (f *PoolFactory) IsEventSupported(topic common.Hash) bool {
	return topic == tokenCreatedEventTopic
}

// DecodePoolCreated parses TokenCreated(ts, creator, nonce, token, name, symbol, meta) - all
// non-indexed - into a fresh entity.Pool. The event doesn't carry the quote token, so this makes one
// extra getTokenV8(token) call to resolve it; StaticExtra is seeded here (immutable metadata the
// tracker never refetches) but Extra is left empty for the tracker's first cycle to fill, same as the
// pool_list_updater.go path.
func (f *PoolFactory) DecodePoolCreated(event ethtypes.Log) (*entity.Pool, error) {
	unpacked, err := portalABI.Events["TokenCreated"].Inputs.Unpack(event.Data)
	if err != nil {
		return nil, err
	}
	if len(unpacked) < 4 {
		return nil, ErrInvalidEvent
	}
	tokenAddress, ok := unpacked[3].(common.Address)
	if !ok {
		return nil, ErrInvalidEvent
	}

	var stateResult tokenStateV8Result
	if _, err := f.ethrpcClient.NewRequest().SetContext(context.Background()).
		AddCall(&ethrpc.Call{
			ABI:    portalABI,
			Target: f.config.PortalAddress,
			Method: "getTokenV8",
			Params: []any{tokenAddress},
		}, []any{&struct{ *tokenStateV8Result }{&stateResult}}).
		Call(); err != nil {
		return nil, err
	}

	quoteTokenRaw := strings.ToLower(stateResult.QuoteTokenAddress.Hex())
	quoteToken := valueobject.ZeroToWrappedLower(quoteTokenRaw, f.config.ChainID)
	token := valueobject.WrapNativeLower(strings.ToLower(tokenAddress.Hex()), f.config.ChainID)

	staticExtraBytes, err := json.Marshal(StaticExtra{
		PortalAddress: f.config.PortalAddress,
	})
	if err != nil {
		return nil, err
	}

	return &entity.Pool{
		Address:     token,
		Exchange:    f.config.DexID,
		Type:        DexType,
		Timestamp:   time.Now().Unix(),
		Reserves:    entity.PoolReserves{"0", "0"},
		StaticExtra: string(staticExtraBytes),
		Tokens: []*entity.PoolToken{
			{Address: quoteToken, Swappable: true},
			{Address: token, Swappable: true},
		},
		BlockNumber: event.BlockNumber,
	}, nil
}

// DecodePoolAddressesFromFactoryLog routes a log emitted on the Portal proxy to the pool (token
// address) it concerns, so pool-service's event-driven refresh (ClassifyLogs, keyed on this method's
// presence via a structural interface check - not part of poolfactory.IPoolFactoryDecoder itself)
// triggers a tracker refresh on every trade instead of only on the interval. This is what closes the
// gap described in pool_tracker.go: swaps happen via delegatecall through the single Portal proxy
// address, so a pool's own token address never itself emits a log the way most AMM pools do.
//
// Not gated by IsEventSupported (that method is only consulted by the new-pool-discovery path,
// GetNewPoolsFromLogs, which must not attempt to decode a non-TokenCreated log as one) - this method
// independently checks topic0 and returns an error for anything it doesn't recognize, which
// pool-service logs at Debug and skips.
func (f *PoolFactory) DecodePoolAddressesFromFactoryLog(_ context.Context, event ethtypes.Log) ([]string, error) {
	if len(event.Topics) == 0 {
		return nil, ErrInvalidEvent
	}

	var tokenAddress common.Address
	switch event.Topics[0] {
	case tokenCreatedEventTopic:
		unpacked, err := portalABI.Events["TokenCreated"].Inputs.Unpack(event.Data)
		if err != nil {
			return nil, err
		}
		if len(unpacked) < 4 {
			return nil, ErrInvalidEvent
		}
		addr, ok := unpacked[3].(common.Address)
		if !ok {
			return nil, ErrInvalidEvent
		}
		tokenAddress = addr
	case flapTokenCirculatingSupplyChangedEventTopic:
		unpacked, err := portalABI.Events["FlapTokenCirculatingSupplyChanged"].Inputs.Unpack(event.Data)
		if err != nil {
			return nil, err
		}
		if len(unpacked) < 1 {
			return nil, ErrInvalidEvent
		}
		addr, ok := unpacked[0].(common.Address)
		if !ok {
			return nil, ErrInvalidEvent
		}
		tokenAddress = addr
	case launchedToDEXEventTopic:
		unpacked, err := portalABI.Events["LaunchedToDEX"].Inputs.Unpack(event.Data)
		if err != nil {
			return nil, err
		}
		if len(unpacked) < 1 {
			return nil, ErrInvalidEvent
		}
		addr, ok := unpacked[0].(common.Address)
		if !ok {
			return nil, ErrInvalidEvent
		}
		tokenAddress = addr
	default:
		return nil, ErrInvalidEvent
	}

	token := valueobject.WrapNativeLower(strings.ToLower(tokenAddress.Hex()), f.config.ChainID)
	return []string{token}, nil
}
