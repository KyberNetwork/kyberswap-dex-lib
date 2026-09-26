package slyngfun

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var _ = poolfactory.RegisterFactoryC(DexType, NewPoolFactoryDecoder)

// PoolFactoryDecoder turns the launchpad's TokenCreated into a pool from the log alone. Every curve
// lives inside the one Launchpad contract, so the pool is keyed by the token it sells: that address
// is unique, it is what buy() and sell() take, and it is what the trade events index. Curve state
// is left to PoolTracker, which pool-service runs as soon as the pool is saved.
type PoolFactoryDecoder struct {
	config *Config
}

func NewPoolFactoryDecoder(cfg *Config) *PoolFactoryDecoder {
	return &PoolFactoryDecoder{config: cfg}
}

func (d *PoolFactoryDecoder) IsEventSupported(hash common.Hash) bool {
	return hash == tokenCreatedEvent.ID
}

func (d *PoolFactoryDecoder) SupportedEventTopics() []common.Hash {
	return []common.Hash{tokenCreatedEvent.ID}
}

func (d *PoolFactoryDecoder) DecodePoolCreated(event types.Log) (*entity.Pool, error) {
	if !strings.EqualFold(event.Address.Hex(), d.config.Launchpad) {
		return nil, nil
	}
	if len(event.Topics) != 4 || event.Topics[0] != tokenCreatedEvent.ID {
		return nil, ErrMalformedLog
	}

	// TokenCreated(address indexed token, address indexed creator, address indexed quote,
	//              string name, string symbol, uint256 lockupSeconds, uint256 creatorUnlockAt,
	//              uint256 graduationTarget)
	values, err := tokenCreatedEvent.Inputs.NonIndexed().Unpack(event.Data)
	if err != nil {
		return nil, err
	}
	if len(values) != 5 {
		return nil, ErrMalformedLog
	}
	graduationTarget, ok := values[4].(*big.Int)
	if !ok || graduationTarget.Sign() < 0 {
		return nil, ErrMalformedLog
	}
	target, overflow := uint256.FromBig(graduationTarget)
	if overflow {
		return nil, ErrMalformedLog
	}

	token := common.BytesToAddress(event.Topics[1].Bytes())
	quote := common.BytesToAddress(event.Topics[3].Bytes())

	// The launchpad prices an ETH coin against address(0) and moves native ETH for it; the
	// aggregator lists it under the wrapped native so the router can wrap and unwrap around it.
	isNativeQuote := valueobject.IsNativeOrZeroAddr(quote)
	if isNativeQuote {
		quote = common.HexToAddress(valueobject.WrappedNativeMap[d.config.ChainID])
	}

	staticExtraBytes, err := json.Marshal(StaticExtra{
		Launchpad:        strings.ToLower(d.config.Launchpad),
		IsNativeQuote:    isNativeQuote,
		GraduationTarget: target,
	})
	if err != nil {
		return nil, err
	}

	return &entity.Pool{
		Address:   hexutil.Encode(token[:]),
		Exchange:  d.config.DexID,
		Type:      DexType,
		Timestamp: time.Now().Unix(),
		Reserves:  entity.PoolReserves{"0", "0"},
		Tokens: []*entity.PoolToken{
			{Address: hexutil.Encode(quote[:]), Swappable: true},
			{Address: hexutil.Encode(token[:]), Swappable: true},
		},
		StaticExtra: string(staticExtraBytes),
		BlockNumber: event.BlockNumber,
	}, nil
}

// DecodePoolAddressesFromFactoryLog routes the launchpad's own trade logs back to the curve they
// belong to. Unlike a per-curve launch contract, this launchpad emits Bought, Sold and Graduated
// itself, with the token as the first indexed topic, so a pool-service watching the factory
// address can refresh exactly the curve that moved.
func (d *PoolFactoryDecoder) DecodePoolAddressesFromFactoryLog(_ context.Context, log types.Log) ([]string, error) {
	if !strings.EqualFold(log.Address.Hex(), d.config.Launchpad) || len(log.Topics) < 2 {
		return nil, nil
	}
	switch log.Topics[0] {
	case boughtEvent.ID, soldEvent.ID, graduatedEvent.ID:
		token := common.BytesToAddress(log.Topics[1].Bytes())
		return []string{hexutil.Encode(token[:])}, nil
	default:
		return nil, nil
	}
}
