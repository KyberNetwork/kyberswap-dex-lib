package kuruob

import (
	"context"
	"math"
	"strconv"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	orderbook "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/order-book"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
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

	var l2BookData []byte
	var vaultParams VaultParamsRPC
	resp, err := t.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    orderBookABI,
		Target: p.Address,
		Method: "getL2Book0",
		Params: []any{uint32(maxPriceLevels), uint32(maxPriceLevels)},
	}, []any{&l2BookData}).AddCall(&ethrpc.Call{
		ABI:    orderBookABI,
		Target: p.Address,
		Method: "getVaultParams",
	}, []any{&vaultParams}).TryBlockAndAggregate()
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to aggregate RPC requests")
		return entity.Pool{}, err
	}

	var extra orderbook.Extra
	var staticExtra StaticExtra
	_ = json.Unmarshal([]byte(p.StaticExtra), &staticExtra)

	var tmp, tmp2 uint256.Int
	offset := 32 // start reading after block number
	pricePrecision, sizePrecision := math.Pow10(staticExtra.PricePrecision), math.Pow10(staticExtra.SizePrecision)
	for i := range extra.LevelsFrom {
		book := make([]orderbook.Level, 1, 1+len(l2BookData)/32/4*3/5) // first level == min trade == 0
		for offset < len(l2BookData) {
			if tmp.SetBytes(l2BookData[offset : offset+32]).IsZero() {
				offset += 32
				break
			}
			price, size := tmp.Float64(), tmp.SetBytes(l2BookData[offset+32:offset+64]).Float64()/sizePrecision
			if i == 0 {
				price = price / pricePrecision
			} else {
				price = pricePrecision / price
				size *= price
			}
			offset += 64
			book = append(book, orderbook.Level{size, price})
		}
		extra.LevelsFrom[i] = book
	}

	if vaultParams.VaultBidOrderSize.Sign() > 0 && vaultParams.KuruAmmVault != valueobject.AddrZero {
		spread := vaultParams.Spread.Uint64() / 10
		currentSize := [2]*uint256.Int{uint256.MustFromBig(vaultParams.VaultBidOrderSize),
			uint256.MustFromBig(vaultParams.VaultAskOrderSize)}
		bidPartiallyFilledSize := uint256.MustFromBig(vaultParams.BidPartiallyFilledSize)
		askPartiallyFilledSize := uint256.MustFromBig(vaultParams.AskPartiallyFilledSize)
		firstOrderSize := [2]*uint256.Int{bidPartiallyFilledSize.Sub(currentSize[0], bidPartiallyFilledSize),
			askPartiallyFilledSize.Sub(currentSize[1], askPartiallyFilledSize)}

		currentPrice := [2]*uint256.Int{uint256.MustFromBig(vaultParams.VaultBestBid),
			uint256.MustFromBig(vaultParams.VaultBestAsk)}
		spreadPlus1k := tmp.SetUint64(spread + 1000)
		spreadPlus2k := tmp2.SetUint64(spread + 2000)

		var ammLevels [2][]orderbook.Level
		for from := range 2 {
			ammLevels[from] = make([]orderbook.Level, 0, (maxPriceLevels+1)/2)
			for i := range maxPriceLevels {
				if currentPrice[from].IsZero() {
					break
				}
				size := currentSize[from]
				if i == 0 {
					size = firstOrderSize[from]
				}
				price, sizeF := currentPrice[from].Float64(), size.Float64()/sizePrecision
				if from == 0 {
					price = price / 1e18
				} else {
					price = 1e18 / price
					sizeF *= price
				}
				ammLevels[from] = append(ammLevels[from], orderbook.Level{sizeF, price})
				if from == 0 {
					// Next bid price = currentPrice * 1000 / (1000 + spreadConstant)
					currentPrice[0].MulDivOverflow(currentPrice[0], big256.U1000, spreadPlus1k)
					// Next bid size = currentSize * (2000 + spreadConstant) / 2000
					currentSize[0].MulDivOverflow(currentSize[0], spreadPlus2k, big256.U2000)
				} else {
					// Next ask price = currentPrice * (1000 + spreadConstant) / 1000
					if _, overflow := currentPrice[1].MulDivOverflow(currentPrice[1], spreadPlus1k,
						big256.U1000); overflow {
						break
					}
					// Next ask size = currentSize * 2000 / (2000 + spreadConstant)
					currentSize[1].MulDivOverflow(currentSize[1], big256.U2000, spreadPlus2k)
				}
			}
		}

		// Merge sort ammLevels into orderbook.Levels
		var mergedLevels [2][]orderbook.Level
		for from := range 2 {
			lenManualLevels, lenAmmLevels := len(extra.LevelsFrom[from]), len(ammLevels[from])
			mergedLevels[from] = make([]orderbook.Level, lenManualLevels+lenAmmLevels)
			var i, j int
			for i < lenManualLevels && j < lenAmmLevels {
				if i == 0 || extra.LevelsFrom[from][i].Price() > ammLevels[from][j].Price() {
					mergedLevels[from][i+j] = extra.LevelsFrom[from][i]
					i++
				} else {
					mergedLevels[from][i+j] = ammLevels[from][j]
					j++
				}
			}
			extra.LevelsFrom[from] = append(append(mergedLevels[from][:i+j],
				extra.LevelsFrom[from][i:]...), ammLevels[from][j:]...)
		}
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to marshal extra data")
		return entity.Pool{}, err
	}

	var reserve0, reserve1 float64
	for _, level := range extra.LevelsFrom[1] {
		reserve0 += level.Size() * level.Price()
	}
	for _, level := range extra.LevelsFrom[0] {
		reserve1 += level.Size() * level.Price()
	}

	p.Reserves = entity.PoolReserves{strconv.FormatFloat(reserve0*math.Pow10(int(p.Tokens[0].Decimals)), 'f', 0, 64),
		strconv.FormatFloat(reserve1*math.Pow10(int(p.Tokens[1].Decimals)), 'f', 0, 64)}
	p.Extra = string(extraBytes)
	p.BlockNumber = resp.BlockNumber.Uint64()

	l.Info("Finish updating state of pool")
	return p, nil
}
