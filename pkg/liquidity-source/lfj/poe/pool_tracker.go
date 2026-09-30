package poe

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

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

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	logger.Infof("getting new pool state for %v", p.Address)

	tokenX := common.HexToAddress(p.Tokens[0].Address)
	tokenY := common.HexToAddress(p.Tokens[1].Address)
	key := crypto.Keccak256Hash(append(tokenX.Bytes(), tokenY.Bytes()...))

	var (
		oracle   common.Address
		reserves struct {
			AmountX *uint256.Int
			AmountY *uint256.Int
		}
	)

	req := t.ethrpcClient.R().SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: p.Address,
			Method: poolMethodGetOracle,
		}, []any{&oracle}).
		AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: p.Address,
			Method: poolMethodGetBalances,
		}, []any{&reserves})
	_, err := req.Aggregate()
	if err != nil {
		return p, err
	}

	var oracleData struct {
		Price   *uint256.Int
		FeeHbps *uint256.Int
		Alpha   *uint256.Int
		Expiry  *uint256.Int
	}

	req = t.ethrpcClient.R().SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    oracleABI,
			Target: oracle.String(),
			Method: "getData",
			Params: []any{key},
		}, []any{&oracleData})
	resp, err := req.Aggregate()
	if err != nil {
		return p, err
	}

	extra := Extra{
		Oracle:  oracle.String(),
		Price:   oracleData.Price,
		FeeHbps: oracleData.FeeHbps,
		Alpha:   oracleData.Alpha,
		Expiry:  oracleData.Expiry.Uint64(),
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves{
		reserves.AmountX.Dec(),
		reserves.AmountY.Dec(),
	}
	p.BlockNumber = resp.BlockNumber.Uint64()
	p.Timestamp = time.Now().Unix()

	logger.Infof("finished getting pool state for %v", p.Address)

	return p, nil
}
