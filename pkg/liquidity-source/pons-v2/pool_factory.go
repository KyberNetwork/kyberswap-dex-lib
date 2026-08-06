package ponsv2

import (
	"context"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// tokenLaunchedLog mirrors Factory.TokenLaunched's decoded fields (only the
// ones this integration needs; deployer and launchConfigId are dropped).
type tokenLaunchedLog struct {
	Token     common.Address
	Curve     common.Address
	PairToken common.Address
}

// PoolFactoryDecoder decodes PonsV2LaunchFactory's TokenLaunched event into a
// pool, purely from log data -- no RPC calls. It's the single source of truth
// for "TokenLaunched log -> pool", reused both for live discovery (registered
// under pool-service's `dependencies:` config) and for historical backfill
// (driven by poolfactory.FilterLogsBackfiller). Reserves and curve-config
// fields (fee/tax/reserved-tokens/graduated) are intentionally left as the
// zero value here -- PoolTracker.GetNewPoolState fills them in on the first
// tracker pass, which pool-service triggers immediately after a new pool is
// saved.
type PoolFactoryDecoder struct {
	config *Config
}

var _ = poolfactory.RegisterFactoryC(DexType, NewPoolFactoryDecoder)

func NewPoolFactoryDecoder(cfg *Config) *PoolFactoryDecoder {
	return &PoolFactoryDecoder{config: cfg}
}

func (d *PoolFactoryDecoder) IsEventSupported(hash common.Hash) bool {
	return hash == tokenLaunchedEventHash
}

func (d *PoolFactoryDecoder) DecodePoolCreated(event types.Log) (*entity.Pool, error) {
	if !strings.EqualFold(event.Address.Hex(), d.config.Factory) {
		return nil, nil
	}

	launch, err := decodeTokenLaunched(event)
	if err != nil {
		return nil, err
	}

	// isNativeQuote() on-chain is true iff pairToken() == address(0); some
	// deployments instead use the 0xEeee... native-ETH sentinel in the same
	// position, so IsNativeOrZeroAddr covers both without an RPC call.
	quoteAddress := launch.PairToken
	if valueobject.IsNativeOrZeroAddr(launch.PairToken) {
		quoteAddress = common.HexToAddress(valueobject.WrappedNativeMap[d.config.ChainID])
	}

	return &entity.Pool{
		Address:   hexutil.Encode(launch.Curve[:]),
		Exchange:  d.config.DexID,
		Type:      DexType,
		Timestamp: time.Now().Unix(),
		Reserves:  []string{"0", "0"},
		Tokens: []*entity.PoolToken{
			{Address: hexutil.Encode(quoteAddress[:]), Swappable: true},
			{Address: hexutil.Encode(launch.Token[:]), Swappable: true},
		},
	}, nil
}

// DecodePoolAddressesFromFactoryLog satisfies pool-service's locally-extended
// decoder interface (used to route factory-emitted, non-creation logs back to
// a pool address for state updates). PonsV2LaunchFactory only ever emits
// TokenLaunched: curve state changes (Buy/Sell) are emitted by each curve
// contract at its own address, which pool-service's live pipeline already
// routes to pool-state updates without going through the factory decoder.
func (d *PoolFactoryDecoder) DecodePoolAddressesFromFactoryLog(_ context.Context, _ types.Log) ([]string, error) {
	return nil, nil
}

// decodeTokenLaunched decodes topics positionally (token, curve, deployer
// are indexed; pairToken/launchConfigId/graduationThreshold are the
// non-indexed data payload, in that declaration order) rather than by
// reflect-based field-name matching, to avoid ABI-unpack naming pitfalls.
func decodeTokenLaunched(l types.Log) (tokenLaunchedLog, error) {
	if len(l.Topics) != 4 {
		return tokenLaunchedLog{}, ErrInvalidToken
	}

	values, err := factoryABI.Events["TokenLaunched"].Inputs.NonIndexed().Unpack(l.Data)
	if err != nil {
		return tokenLaunchedLog{}, err
	}
	if len(values) == 0 {
		return tokenLaunchedLog{}, ErrInvalidToken
	}
	pairToken, ok := values[0].(common.Address)
	if !ok {
		return tokenLaunchedLog{}, ErrInvalidToken
	}

	return tokenLaunchedLog{
		Token:     common.BytesToAddress(l.Topics[1].Bytes()),
		Curve:     common.BytesToAddress(l.Topics[2].Bytes()),
		PairToken: pairToken,
	}, nil
}
