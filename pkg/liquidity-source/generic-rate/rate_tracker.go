package generic_rate

import (
	"context"
	"fmt"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	skypsm "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/maker/sky-psm"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/maker/spark"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type SwapFunc struct {
	Func     string `json:"func,omitempty"`
	ArgIdxes []int  `json:"argIdxes,omitempty"`
}

type IRateTracker interface {
	GetSwapData(ctx context.Context, p *entity.Pool, blockTimestamp uint64) ([]*uint256.Int, SwapFuncData, error)
}

type (
	SkyPSMRateTracker struct {
		ethrpcClient *ethrpc.Client
	}
)

func (r *SkyPSMRateTracker) GetSwapData(
	ctx context.Context,
	p *entity.Pool,
	blockTimestamp uint64,
) ([]*uint256.Int, SwapFuncData, error) {
	var extra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &extra); err != nil {
		return nil, nil, err
	}

	var rate *big.Int
	req := r.ethrpcClient.R().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    skyPSMABI,
		Target: extra.RateProvider,
		Method: "getConversionRate",
		Params: []interface{}{big.NewInt(int64(blockTimestamp + spark.Blocktime))},
	}, []interface{}{&rate})
	resp, err := req.Aggregate()
	if err != nil {
		fmt.Println(resp)
		return nil, nil, err
	}

	usdcPrecision := bignumber.TenPowInt(p.Tokens[0].Decimals)
	usdsPrecision := bignumber.TenPowInt(p.Tokens[1].Decimals)
	susdsPrecision := bignumber.TenPowInt(p.Tokens[2].Decimals)

	args := []*uint256.Int{
		uint256.MustFromBig(rate),
		uint256.MustFromBig(usdcPrecision),
		uint256.MustFromBig(usdsPrecision),
		uint256.MustFromBig(susdsPrecision),
	}

	return args, SwapFuncData{
		0: {
			1: SwapFunc{Func: skypsm.OneToOne, ArgIdxes: []int{1, 2}},
			2: SwapFunc{Func: skypsm.ToSUSDS, ArgIdxes: []int{1}},
		},
		1: {
			0: SwapFunc{Func: skypsm.OneToOne, ArgIdxes: []int{2, 1}},
			2: SwapFunc{Func: skypsm.ToSUSDS, ArgIdxes: []int{2}},
		},
		2: {
			0: SwapFunc{Func: skypsm.FromSUSDS, ArgIdxes: []int{0, 1}},
			1: SwapFunc{Func: skypsm.FromSUSDS, ArgIdxes: []int{0, 2}},
		},
	}, nil
}
