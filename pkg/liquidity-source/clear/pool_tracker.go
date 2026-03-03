package clear

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	logger.Infof("[Clear] Start getting new state of pool: %v", p.Address)
	if len(p.Tokens) < 2 {
		return entity.Pool{}, ErrPoolNotFound
	}

	req := d.ethrpcClient.NewRequest().SetContext(ctx)
	poolAddr := common.HexToAddress(p.Address)
	tokens := lo.Map(p.Tokens, func(token *entity.PoolToken, _ int) common.Address {
		return common.HexToAddress(token.Address)
	})
	iouTokens := make([]common.Address, len(p.Tokens))
	rates := make([][]AmtInOut, len(p.Tokens))
	output := make([][]*big.Int, len(p.Tokens))
	tokenBalances := make([]*big.Int, len(p.Tokens))
	for i, token := range p.Tokens {
		req.AddCall(&ethrpc.Call{
			ABI:    clearVaultABI,
			Target: p.Address,
			Method: methodIouOf,
			Params: []any{tokens[i]},
		}, []any{&iouTokens[i]}).AddCall(&ethrpc.Call{
			ABI:    clearVaultABI,
			Target: p.Address,
			Method: methodTokenAssets,
			Params: []any{tokens[i]},
		}, []any{&tokenBalances[i]})
		rates[i] = make([]AmtInOut, len(p.Tokens))
		output[i] = make([]*big.Int, len(p.Tokens))
		for j := range p.Tokens {
			if i == j {
				continue
			}
			amountIn := bignumber.TenPowInt(token.Decimals)
			rates[i][j][0] = uint256.MustFromBig(amountIn)
			req.AddCall(&ethrpc.Call{
				ABI:    clearSwapABI,
				Target: d.config.SwapAddress,
				Method: methodPreviewSwap,
				Params: []any{poolAddr, tokens[i], tokens[j], amountIn, true},
			}, []any{&output[i][j]})
		}
	}

	if _, err := req.TryAggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("[Clear] failed to call previewSwap")
		return entity.Pool{}, nil
	}

	hasSwap := make([]bool, len(p.Tokens))
	for i, o := range output {
		for j, o := range o {
			if o == nil {
				rates[i][j][0] = nil
			} else {
				rates[i][j][1] = uint256.MustFromBig(o)
				hasSwap[j] = true
			}
		}
	}
	extra := Extra{
		SwapAddress: d.config.SwapAddress,
		IOUs: lo.Map(iouTokens,
			func(iouToken common.Address, _ int) string { return hexutil.Encode(iouToken[:]) }),
		Rates: rates,
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("[Clear] failed to marshal extra")
		return entity.Pool{}, err
	}

	p.Reserves = lo.Map(tokenBalances, func(bal *big.Int, i int) string {
		if hasSwap[i] {
			return bal.String()
		}
		return "0"
	})
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	logger.Infof("[Clear] Finish getting new state of pool: %v", p.Address)
	return p, nil
}
