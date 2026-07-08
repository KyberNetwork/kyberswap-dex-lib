package smoothy

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var addressMask = bignumber.NewBig("1461501637330902918203684832716283019655932542975") // 2^160 -1

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

const (
	decimalMultiplierOffset = 160 + 41
	decimalMultiplierMask   = 0x1F
)

func getDecimalMultiplier(info *big.Int) uint8 {
	shifted := new(big.Int).Rsh(info, decimalMultiplierOffset)
	return uint8(shifted.Uint64() & decimalMultiplierMask)
}

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

type TokenStats struct {
	SoftWeight *big.Int
	HardWeight *big.Int
	Balance    *big.Int
	Decimals   *big.Int
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	logger.Infof("getting new pool state for %s", p.Address)
	defer logger.Infof("finished getting pool state for %s", p.Address)

	var (
		swapFee      *big.Int
		adminFeePct  *big.Int
		totalBalance *big.Int
		nTokens      *big.Int
	)

	req := t.ethrpcClient.R().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    smoothyV1ABI,
		Target: p.Address,
		Method: "_swapFee",
	}, []any{&swapFee}).AddCall(&ethrpc.Call{
		ABI:    smoothyV1ABI,
		Target: p.Address,
		Method: "_adminFeePct",
	}, []any{&adminFeePct}).AddCall(&ethrpc.Call{
		ABI:    smoothyV1ABI,
		Target: p.Address,
		Method: "_totalBalance",
	}, []any{&totalBalance}).AddCall(&ethrpc.Call{
		ABI:    smoothyV1ABI,
		Target: p.Address,
		Method: "_ntokens",
	}, []any{&nTokens})
	resp, err := req.Aggregate()
	if err != nil {
		logger.Errorf("failed to get pool parameters: %v", err)
		return p, err
	}

	numTokens := int(nTokens.Int64())
	tokenStats := make([]TokenStats, numTokens)
	packedInfos := make([]*big.Int, numTokens)
	req = t.ethrpcClient.R().SetContext(ctx).SetBlockNumber(resp.BlockNumber)
	for i := 0; i < numTokens; i++ {
		req.AddCall(&ethrpc.Call{
			ABI:    smoothyV1ABI,
			Target: p.Address,
			Method: "getTokenStats",
			Params: []any{big.NewInt(int64(i))},
		}, []any{&tokenStats[i]}).AddCall(&ethrpc.Call{
			ABI:    smoothyV1ABI,
			Target: p.Address,
			Method: "_tokenInfos",
			Params: []any{big.NewInt(int64(i))},
		}, []any{&packedInfos[i]})
	}
	_, err = req.Aggregate()
	if err != nil {
		logger.Errorf("failed to get token stats: %v", err)
		return p, err
	}

	tokenInfos := make([]TokenInfo, numTokens)
	reserves := make(entity.PoolReserves, numTokens)
	tokens := make([]*entity.PoolToken, numTokens)

	for i := 0; i < numTokens; i++ {
		stats := tokenStats[i]
		decimals := uint8(stats.Decimals.Uint64())
		decimalMultiplier := getDecimalMultiplier(packedInfos[i])

		tokenAddress := strings.ToLower(common.BigToAddress(new(big.Int).And(packedInfos[i], addressMask)).Hex())

		tokenInfos[i] = TokenInfo{
			SoftWeight:         number.SetFromBig(stats.SoftWeight),
			HardWeight:         number.SetFromBig(stats.HardWeight),
			DecimalMulitiplier: decimalMultiplier,
			Balance:            number.SetFromBig(stats.Balance),
		}

		tokens[i] = &entity.PoolToken{
			Address:   tokenAddress,
			Decimals:  decimals,
			Swappable: true,
		}

		reserves[i] = stats.Balance.String()
	}

	extra := Extra{
		SwapFee:      number.SetFromBig(swapFee),
		AdminFeePct:  number.SetFromBig(adminFeePct),
		TotalBalance: number.SetFromBig(totalBalance),
		TokenInfos:   tokenInfos,
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = reserves
	p.Tokens = tokens
	p.Timestamp = time.Now().Unix()
	p.BlockNumber = resp.BlockNumber.Uint64()

	return p, nil
}
