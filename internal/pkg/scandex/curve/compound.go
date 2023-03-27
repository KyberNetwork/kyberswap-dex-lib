package curve

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/abis"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/constant"
	curveCompound "github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/core/curve-compound"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/repository"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/logger"

	errorsPkg "github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/core/errors"

	"github.com/ethereum/go-ethereum/common"
)

func (t *Curve) getCompoundExtra(ctx context.Context, pool entity.Pool) (interface{}, error) {

	if len(t.properties.AddressesFromProvider) == 0 {
		return nil, errorsPkg.ErrProvidersNotSupported
	}
	mainRegistry := t.properties.AddressesFromProvider[0]
	var calls []*repository.CallParams
	var a, swapFee, adminFee *big.Int
	//the get_rates method's response size is fixed = 8
	var rates8 [8]*big.Int
	calls = append(calls, &repository.CallParams{
		ABI:    abis.CurveCompound,
		Target: pool.Address,
		Method: "A",
		Params: nil,
		Output: &a,
	})

	calls = append(calls, &repository.CallParams{
		ABI:    abis.CurveCompound,
		Target: pool.Address,
		Method: "fee",
		Params: nil,
		Output: &swapFee,
	})

	calls = append(calls, &repository.CallParams{
		ABI:    abis.CurveCompound,
		Target: pool.Address,
		Method: "admin_fee",
		Params: nil,
		Output: &adminFee,
	})

	calls = append(calls, &repository.CallParams{
		ABI:    abis.CurveMainRegistry,
		Target: mainRegistry,
		Method: "get_rates",
		Params: []interface{}{common.HexToAddress(pool.Address)},
		Output: &rates8,
	})

	if err := t.scanService.MultiCall(ctx, calls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return nil, err
	}

	var rates []string
	for i := 0; i < len(pool.Tokens); i++ {
		if rates8[i] == constant.Zero {
			return nil, ErrOneTokenHasNoRate
		} else {
			rates = append(rates, rates8[i].String())
		}
	}
	extra := curveCompound.Extra{
		A:        a.String(),
		SwapFee:  swapFee.String(),
		AdminFee: adminFee.String(),
		Rates:    rates,
	}
	return extra, nil
}

func (t *Curve) getCompoundPoolReserves(ctx context.Context, pool entity.Pool) ([]*big.Int, error) {

	nTokens := len(pool.Tokens)
	reserves := make([]*big.Int, 2*nTokens)

	tryCalls := make([]*repository.TryCallParams, 0)
	for j := range pool.Tokens {
		tryCalls = append(tryCalls, &repository.TryCallParams{
			ABI:    abis.CurveAave,
			Target: pool.Address,
			Method: "balances",
			Params: []interface{}{big.NewInt(int64(j))},
			Output: &reserves[2*j],
		})

		tryCalls = append(tryCalls, &repository.TryCallParams{
			ABI:    abis.CurveAaveV1,
			Target: pool.Address,
			Method: "balances",
			Params: []interface{}{big.NewInt(int64(j))},
			Output: &reserves[2*j+1],
		})
	}
	if err := t.scanService.TryAggregateForce(ctx, false, tryCalls); err != nil {
		logger.Errorf("failed to process multicall for getCompoundPoolReserves(%v), err: %v", pool.Address, err)
		return nil, err
	}

	res := make([]*big.Int, 0, nTokens)
	for i := 0; i < nTokens; i++ {
		if reserves[2*i] != nil {
			res = append(res, reserves[2*i])
		} else if reserves[2*i+1] != nil {
			res = append(res, reserves[2*i+1])
		} else {
			return nil, ErrCanNotGetBalances
		}
	}

	return res, nil
}
