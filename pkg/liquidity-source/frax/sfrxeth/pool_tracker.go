package sfrxeth

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	frax_common "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/frax/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	totalSupply, totalAssets, extra, blockNumber, err := getState(
		ctx, p.Address, p.Tokens[1].Address, t.ethrpcClient,
	)
	if err != nil {
		return p, err
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Reserves = entity.PoolReserves{totalAssets.String(), totalSupply.String()}
	p.Extra = string(extraBytes)
	p.BlockNumber = blockNumber
	p.Timestamp = time.Now().Unix()

	return p, nil
}

func getState(
	ctx context.Context,
	minterAddress string,
	sfrxETHAddress string,
	ethrpcClient *ethrpc.Client,
) (*big.Int, *big.Int, PoolExtra, uint64, error) {
	var (
		submitPaused bool
		totalSupply  *big.Int
		totalAssets  *big.Int
	)

	calls := ethrpcClient.NewRequest().SetContext(ctx)
	calls.AddCall(&ethrpc.Call{
		ABI:    frax_common.FrxETHMinterABI,
		Target: minterAddress,
		Method: minterMethodSubmitPaused,
	}, []interface{}{&submitPaused})
	calls.AddCall(&ethrpc.Call{
		ABI:    frax_common.SfrxETHABI,
		Target: sfrxETHAddress,
		Method: SfrxETHMethodTotalAssets,
	}, []interface{}{&totalAssets})
	calls.AddCall(&ethrpc.Call{
		ABI:    frax_common.SfrxETHABI,
		Target: sfrxETHAddress,
		Method: SfrxETHMethodTotalSupply,
	}, []interface{}{&totalSupply})

	resp, err := calls.Aggregate()
	if err != nil {
		return nil, nil, PoolExtra{}, 0, err
	}

	if resp.BlockNumber == nil {
		resp.BlockNumber = big.NewInt(0)
	}

	poolExtra := PoolExtra{
		SubmitPaused: submitPaused,
	}

	return totalSupply, totalAssets, poolExtra, resp.BlockNumber.Uint64(), nil
}
