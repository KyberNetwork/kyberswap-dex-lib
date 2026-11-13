package lglclob

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ poolpkg.GetNewPoolStateParams,
) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       t.config.DexID,
	})
	l.Info("Start getting new state")

	var last Extra
	_ = json.Unmarshal([]byte(p.Extra), &last)
	now := time.Now()
	measure := measureDue(last, now)
	state, err := readState(ctx, t.ethrpcClient, common.HexToAddress(p.Address),
		common.HexToAddress(t.config.HelperAddress), measure)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to read the pool's state")
		return entity.Pool{}, err
	}

	// The market maker and its quote are measured now and then, and kept in between.
	extra := Extra{OrderBook: state.book, MakerQuoteGas: last.MakerQuoteGas, MeasuredAt: last.MeasuredAt,
		Maker: last.Maker}
	if measure {
		extra.MakerQuoteGas, extra.MeasuredAt, extra.Maker = state.makerQuoteGas(), now.Unix(), ""
		if state.hasMaker {
			maker, err := readMaker(ctx, t.ethrpcClient, common.HexToAddress(p.Address))
			if err != nil {
				l.WithFields(logger.Fields{
					"error": err,
				}).Error("failed to read the pool's market maker")
				return entity.Pool{}, err
			}
			extra.Maker = strings.ToLower(maker.Hex())
		}
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to marshal extra data")
		return entity.Pool{}, err
	}

	var staticExtra StaticExtra
	_ = json.Unmarshal([]byte(p.StaticExtra), &staticExtra)
	var reserveX, reserveY uint256.Int
	for i, share := range state.book.Bids.ArrayShares {
		reserveY.Add(&reserveY, reserveX.Mul(share, state.book.Bids.ArrayPrices[i]))
	}
	reserveX.Clear()
	for _, share := range state.book.Asks.ArrayShares {
		reserveX.Add(&reserveX, share)
	}

	p.Reserves = entity.PoolReserves{reserveX.Mul(&reserveX, staticExtra.ScalingFactorX).String(),
		reserveY.Mul(&reserveY, staticExtra.ScalingFactorY).String()}
	p.Extra = string(extraBytes)
	p.BlockNumber = state.block
	p.Timestamp = now.Unix()

	l.Info("Finish updating state of pool")
	return p, nil
}

// readMaker reads pool's market maker.
func readMaker(ctx context.Context, client *ethrpc.Client, pool common.Address) (common.Address, error) {
	data, err := client.GetETHClient().CallContract(ctx, ethereum.CallMsg{To: &pool, Data: makerSelector}, nil)
	if err != nil {
		return common.Address{}, err
	}
	if len(data) != common.HashLength {
		return common.Address{}, ErrUnexpectedMaker
	}
	return common.BytesToAddress(data), nil
}
