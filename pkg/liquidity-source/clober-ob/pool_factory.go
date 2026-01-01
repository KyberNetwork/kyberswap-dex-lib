package cloberob

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	abis "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/clober-ob/abi"
	cloberlib "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/clober-ob/libraries"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var _ = poolfactory.RegisterFactoryC(DexType, NewPoolFactory)

type PoolFactory struct {
	config              *Config
	bookCreatedEventIds map[common.Hash]struct{}
}

func NewPoolFactory(config *Config) *PoolFactory {
	return &PoolFactory{
		config: config,
		bookCreatedEventIds: map[common.Hash]struct{}{
			bookManagerABI.Events["Open"].ID: {},
		},
	}
}

func (f *PoolFactory) DecodePoolCreated(event types.Log) (*entity.Pool, error) {
	p, err := bookManagerFilterer.ParseOpen(event)
	if err != nil {
		return nil, err
	}

	return f.newPool(p, event.BlockNumber)
}

func (f *PoolFactory) IsEventSupported(event common.Hash) bool {
	_, ok := f.bookCreatedEventIds[event]
	return ok
}

func (f *PoolFactory) newPool(p *abis.BookManagerOpen, blockNumber uint64) (*entity.Pool, error) {
	staticExtraBytes, err := json.Marshal(StaticExtra{
		Base:        p.Base,
		Quote:       p.Quote,
		UnitSize:    p.UnitSize,
		MakerPolicy: cloberlib.FeePolicy(p.MakerPolicy.Uint64()),
		TakerPolicy: cloberlib.FeePolicy(p.TakerPolicy.Uint64()),
		Hooks:       p.Hooks,
		BookManager: f.config.BookManager,
	})
	if err != nil {
		return nil, err
	}

	return &entity.Pool{
		Address:   p.Id.String(),
		Exchange:  GetExchangeByHook(p.Hooks),
		Type:      DexType,
		Timestamp: time.Now().Unix(),
		Reserves:  entity.PoolReserves{"0", "0"},
		Tokens: []*entity.PoolToken{
			{
				Address:   valueobject.ZeroToWrappedLower(p.Base.String(), f.config.ChainId),
				Swappable: true,
			},
			{
				Address:   valueobject.ZeroToWrappedLower(p.Quote.String(), f.config.ChainId),
				Swappable: true,
			},
		},
		Extra:       "{}",
		StaticExtra: string(staticExtraBytes),
		BlockNumber: blockNumber,
	}, nil
}

func DecodePoolAddress(log types.Log) (string, error) {
	if len(log.Topics) == 0 || eth.IsZeroAddress(log.Address) {
		return "", nil
	}

	switch log.Topics[0] {
	case bookManagerABI.Events["Open"].ID:
		openEvent, err := bookManagerFilterer.ParseOpen(log)
		if err != nil {
			return "", err
		}

		return openEvent.Id.String(), nil

	case bookManagerABI.Events["Make"].ID:
		makeEvent, err := bookManagerFilterer.ParseMake(log)
		if err != nil {
			return "", err
		}

		return makeEvent.BookId.String(), nil

	case bookManagerABI.Events["Take"].ID:
		takeEvent, err := bookManagerFilterer.ParseTake(log)
		if err != nil {
			return "", err
		}

		return takeEvent.BookId.String(), nil

	case bookManagerABI.Events["Claim"].ID:
		event, err := bookManagerFilterer.ParseClaim(log)
		if err != nil {
			return "", err
		}

		bookId, _ := cloberlib.DecodeOrderId(event.OrderId)
		return bookId, nil

	case bookManagerABI.Events["Cancel"].ID:
		event, err := bookManagerFilterer.ParseCancel(log)
		if err != nil {
			return "", err
		}

		bookId, _ := cloberlib.DecodeOrderId(event.OrderId)
		return bookId, nil
	}

	return "", nil
}
