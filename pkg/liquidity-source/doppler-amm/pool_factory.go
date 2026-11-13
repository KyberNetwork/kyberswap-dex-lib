package doppleramm

import (
	"context"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Topic0 of the AMM events whose topic1 is the market id. The AMM is unverified, so the hashes
// come from its live logs: a swap emits the first, liquidity add and remove emit the others.
var marketStateTopics = map[common.Hash]struct{}{
	common.HexToHash("0x07af4e23b8e693b486c551709747254c747c07d7f9b3930a72bb6b0c3ac5af83"): {}, // swap
	common.HexToHash("0xdb56780297c6bee5d6fcac88e28b683fc68cb5d85fea9205cd3ce956e8481e19"): {}, // liquidity add
	common.HexToHash("0x33c06bd624e4eb0d07c026198804af254b48b0a4d48f63a88e7a1e967ab7a983"): {}, // liquidity remove
}

var _ = poolfactory.RegisterFactory(DexType, NewPoolFactory)

// PoolFactory decodes the AMM singleton's MarketCreated logs into pools (live and backfill) and
// routes its market events to the market's pool. It lists both market kinds and sets Exchange by
// module, because pool-service keeps one decoder per factory address: every registration for the
// AMM must behave the same.
type PoolFactory struct {
	cfg    *Config
	amm    common.Address
	warned sync.Map // skipped module/extension pairs already logged
}

func NewPoolFactory(params poolfactory.PoolsDecoderParams[Config]) (*PoolFactory, error) {
	cfg := params.Cfg
	if err := cfg.withDefaults(); err != nil {
		return nil, err
	}
	return &PoolFactory{cfg: cfg, amm: common.HexToAddress(cfg.AMMAddress)}, nil
}

func (f *PoolFactory) IsEventSupported(hash common.Hash) bool {
	return hash == ammABI.Events["MarketCreated"].ID
}

// SupportedEventTopics narrows a backfill scan to MarketCreated.
func (f *PoolFactory) SupportedEventTopics() []common.Hash {
	return []common.Hash{ammABI.Events["MarketCreated"].ID}
}

// DecodePoolAddressesFromFactoryLog maps a swap or liquidity log to its market id, the pool address.
func (f *PoolFactory) DecodePoolAddressesFromFactoryLog(_ context.Context, log types.Log) ([]string, error) {
	if log.Address != f.amm || len(log.Topics) < 2 {
		return nil, nil
	}
	if _, ok := marketStateTopics[log.Topics[0]]; !ok {
		return nil, nil
	}
	return []string{strings.ToLower(hexutil.Encode(log.Topics[1][:]))}, nil
}

type marketCreatedEvent struct {
	MarketID            *big.Int       `abi:"marketId"`
	MarketConfigPointer common.Address `abi:"marketConfigPointer"`
	TokenX              common.Address `abi:"tokenX"`
	TokenY              common.Address `abi:"tokenY"`
	Liquidity           *big.Int       `abi:"liquidity"`
	UniqueKey           *big.Int       `abi:"uniqueKey"`
	LPTokenID           *big.Int       `abi:"lpTokenId"`
	BootstrapX          *big.Int       `abi:"bootstrapX"`
	BootstrapY          *big.Int       `abi:"bootstrapY"`
}

// DecodePoolCreated keeps only markets this source can quote: spot markets (CPMM module, no
// extension) or launch markets (launch module with the FeesLocker extension). The AMM lists
// modules only, so each market's module and extension come from its own event. Any other
// composition may need component payloads that cannot be derived from stored state, so it is
// skipped (nil, nil) with one log line per pair rather than quoted with guessed bytes.
func (f *PoolFactory) DecodePoolCreated(log types.Log) (*entity.Pool, error) {
	if log.Address != f.amm || len(log.Topics) < 4 || log.Topics[0] != ammABI.Events["MarketCreated"].ID {
		return nil, nil
	}
	module := common.BytesToAddress(log.Topics[1].Bytes())
	extension := common.BytesToAddress(log.Topics[2].Bytes())
	var launch bool
	switch {
	case module == common.HexToAddress(f.cfg.SpotModuleAddress) && extension == (common.Address{}):
	case module == common.HexToAddress(f.cfg.LaunchModuleAddress) && extension == common.HexToAddress(f.cfg.FeesLockerAddress):
		launch = true
	default:
		if _, seen := f.warned.LoadOrStore(module.Hex()+extension.Hex(), struct{}{}); !seen {
			logger.WithFields(logger.Fields{"module": module.Hex(), "extension": extension.Hex()}).
				Warn("doppler-amm: skipping market with unsupported module or extension")
		}
		return nil, nil
	}

	var event marketCreatedEvent
	if err := ammABI.UnpackIntoInterface(&event, "MarketCreated", log.Data); err != nil {
		return nil, err
	}
	if event.MarketID == nil || event.TokenX == (common.Address{}) || event.TokenY == (common.Address{}) {
		return nil, ErrUnsupportedMarket
	}

	marketID := hexutil.Encode(common.BigToHash(event.MarketID).Bytes())
	static := StaticExtra{
		MarketID: marketID,
		TokenX:   hexutil.Encode(event.TokenX[:]),
		TokenY:   hexutil.Encode(event.TokenY[:]),
		AMM:      f.cfg.AMMAddress,
		Quoter:   f.cfg.QuoterAddress,
	}
	exchange := valueobject.ExchangeDopplerAMM
	if launch {
		exchange = valueobject.ExchangeDopplerFun
		static.Module = hexutil.Encode(module[:]) // the tracker fills Levels on first refresh
	}
	staticExtra, err := json.Marshal(static)
	if err != nil {
		return nil, err
	}

	// The tracker fills the fees; seed valid JSON so the simulator can load the pool before then.
	extra, err := json.Marshal(Extra{TokenX: hexutil.Encode(event.TokenX[:])})
	if err != nil {
		return nil, err
	}

	return &entity.Pool{
		Address:   marketID,
		Exchange:  string(exchange),
		Type:      DexType,
		Timestamp: time.Now().Unix(),
		Reserves:  entity.PoolReserves{"0", "0"},
		Tokens: []*entity.PoolToken{
			{Address: hexutil.Encode(event.TokenX[:]), Swappable: true},
			{Address: hexutil.Encode(event.TokenY[:]), Swappable: true},
		},
		Extra:       string(extra),
		StaticExtra: string(staticExtra),
		BlockNumber: log.BlockNumber,
	}, nil
}
