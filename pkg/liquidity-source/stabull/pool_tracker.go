package stabull

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/rs/zerolog/log"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
}

// GetNewPoolState updates the pool state by fetching current reserves and parameters
func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool, _ pool.GetNewPoolStateParams) (entity.Pool,
	error) {
	l := log.Ctx(ctx).With().Str("dex", DexType).Str("pool", p.Address).Logger()
	l.Info().Msg("Start getting new state of pool")

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		l.Err(err).Msg("failed to decode StaticExtra data")
		return entity.Pool{}, errors.New("failed to decode StaticExtra")
	}

	var curveRes struct{ Alpha, Beta, Delta, Epsilon, Lambda *big.Int }
	var reserves [2]*big.Int
	var rates [2]*big.Int
	poolAddr := common.HexToAddress(p.Address)
	if _, err := t.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    stabullPoolABI,
		Target: p.Address,
		Method: poolMethodCurve,
	}, []any{&curveRes}).AddCall(&ethrpc.Call{
		ABI:    abi.Erc20ABI,
		Target: p.Tokens[0].Address,
		Method: abi.Erc20BalanceOfMethod,
		Params: []any{poolAddr},
	}, []any{&reserves[0]}).AddCall(&ethrpc.Call{
		ABI:    abi.Erc20ABI,
		Target: p.Tokens[1].Address,
		Method: abi.Erc20BalanceOfMethod,
		Params: []any{poolAddr},
	}, []any{&reserves[1]}).AddCall(&ethrpc.Call{
		ABI:    assimilatorABI,
		Target: staticExtra.Assimilators[0],
		Method: assimilatorMethodGetRate,
	}, []any{&rates[0]}).AddCall(&ethrpc.Call{
		ABI:    assimilatorABI,
		Target: staticExtra.Assimilators[1],
		Method: assimilatorMethodGetRate,
	}, []any{&rates[1]}).TryAggregate(); err != nil {
		l.Err(err).Msg("failed to fetch state from rpc")
		return p, err
	}

	extra := Extra{
		CurveParams: CurveParams{
			Alpha:   uint256.MustFromBig(curveRes.Alpha),
			Beta:    uint256.MustFromBig(curveRes.Beta),
			Delta:   uint256.MustFromBig(curveRes.Delta),
			Epsilon: uint256.MustFromBig(curveRes.Epsilon),
			Lambda:  uint256.MustFromBig(curveRes.Lambda),
		},
		OracleRates: [2]*uint256.Int{uint256.MustFromBig(rates[0]), uint256.MustFromBig(rates[1])},
	}
	// oracleRate = baseRate / quoteRate (scaled by precision)
	extra.OracleRate, _ = new(uint256.Int).MulDivOverflow(extra.OracleRates[0], big256.BONE, extra.OracleRates[1])

	p.Reserves = entity.PoolReserves{reserves[0].String(), reserves[1].String()}
	extraBytes, _ := json.Marshal(extra)
	p.Extra = string(extraBytes)
	p.SwapFee = math.Round(extra.Epsilon.Float64()*1e8/math.Pow(2, 64)) / 1e8
	p.Timestamp = time.Now().Unix()

	l.Info().Msg("Finished getting new state of pool")
	return p, nil
}
