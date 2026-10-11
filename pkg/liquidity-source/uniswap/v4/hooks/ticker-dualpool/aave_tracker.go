package tickerdualpool

import (
	"context"
	_ "embed"
	"math/big"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
)

//go:embed abis/Aave.json
var aaveJSON string
var aaveABI = func() abi.ABI {
	a, err := abi.JSON(strings.NewReader(aaveJSON))
	if err != nil {
		panic(err)
	}
	return a
}()

type reserveData struct {
	Configuration, LiquidityIndex, CurrentLiquidityRate, VariableBorrowIndex, CurrentVariableBorrowRate, CurrentStableBorrowRate *big.Int
	LastUpdateTimestamp                                                                                                          *big.Int
	Id                                                                                                                           uint16
	ATokenAddress, StableDebtTokenAddress, VariableDebtTokenAddress, InterestRateStrategyAddress                                 common.Address
	AccruedToTreasury, Unbacked, IsolationModeTotalDebt                                                                          *big.Int
}

func readAaveReserve(ctx context.Context, p *uniswapv4.HookParam, strategy, asset common.Address, rate *uint256.Int) (*AaveReserve, error) {
	var pool common.Address
	req := p.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(p.BlockNumber).SetOverrides(p.Overrides)
	req.AddCall(&ethrpc.Call{ABI: aaveABI, Target: hexutil.Encode(strategy[:]), Method: "POOL"}, []any{&pool})
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}
	var response struct{ Data reserveData }
	var cash, normalizedDebt *big.Int
	req = p.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(p.BlockNumber).SetOverrides(p.Overrides)
	target := hexutil.Encode(pool[:])
	req.AddCall(&ethrpc.Call{ABI: aaveABI, Target: target, Method: "getReserveData", Params: []any{asset}}, []any{&response})
	req.AddCall(&ethrpc.Call{ABI: aaveABI, Target: target, Method: "getVirtualUnderlyingBalance", Params: []any{asset}}, []any{&cash})
	req.AddCall(&ethrpc.Call{ABI: aaveABI, Target: target, Method: "getReserveNormalizedVariableDebt", Params: []any{asset}}, []any{&normalizedDebt})
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}
	data := response.Data
	var scaledSupply, scaledDebt *big.Int
	req = p.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(p.BlockNumber).SetOverrides(p.Overrides)
	req.AddCall(&ethrpc.Call{ABI: aaveABI, Target: hexutil.Encode(data.ATokenAddress[:]), Method: "scaledTotalSupply"}, []any{&scaledSupply})
	req.AddCall(&ethrpc.Call{ABI: aaveABI, Target: hexutil.Encode(data.VariableDebtTokenAddress[:]), Method: "scaledTotalSupply"}, []any{&scaledDebt})
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}
	for _, v := range []*big.Int{cash, normalizedDebt, scaledSupply, scaledDebt, data.Configuration, data.AccruedToTreasury, data.VariableBorrowIndex} {
		if v == nil || v.Sign() < 0 || v.BitLen() > 256 {
			return nil, ErrInvalidState
		}
	}
	conf := uint256.MustFromBig(data.Configuration)
	bit := func(n uint) bool { return new(uint256.Int).Rsh(conf, n).Uint64()&1 != 0 }
	decimals := new(uint256.Int).Rsh(conf, 48).Uint64() & 255
	if decimals > 77 {
		return nil, ErrInvalidState
	}
	capUnits := new(uint256.Int).Rsh(conf, 116).Uint64() & ((1 << 36) - 1)
	scale := new(uint256.Int).Exp(uint256.NewInt(10), uint256.NewInt(decimals))
	cap := new(uint256.Int)
	if _, over := cap.MulOverflow(scale, uint256.NewInt(capUnits)); over {
		return nil, ErrInvalidState
	}
	supply, err := addChecked(uint256.MustFromBig(scaledSupply), uint256.MustFromBig(data.AccruedToTreasury))
	if err != nil {
		return nil, err
	}
	// Aave accrues the reserve's share of new interest on its next supply/withdraw.
	// Native maxDeposit still uses the stored treasury amount until that mutation.
	factor := new(uint256.Int).Rsh(conf, 64).Uint64() & 65535
	pending, err := pendingTreasuryShares(uint256.MustFromBig(scaledDebt), uint256.MustFromBig(data.VariableBorrowIndex), uint256.MustFromBig(normalizedDebt), rate, factor)
	if err != nil {
		return nil, err
	}
	return &AaveReserve{Cash: uint256.MustFromBig(cash), ScaledSupply: supply, SupplyCap: cap, PendingTreasury: pending, Active: bit(56), Frozen: bit(57), Paused: bit(60)}, nil
}

func pendingTreasuryShares(debt, previous, current, rate *uint256.Int, factor uint64) (*uint256.Int, error) {
	if debt == nil || previous == nil || current == nil || rate == nil || rate.IsZero() || factor > 10000 || current.Cmp(previous) < 0 {
		return nil, ErrInvalidState
	}
	debtAccrued, err := v3Utils.MulDiv(debt, new(uint256.Int).Sub(current, previous), ray)
	if err != nil {
		return nil, err
	}
	product := new(uint256.Int)
	if _, over := product.MulOverflow(debtAccrued, uint256.NewInt(factor)); over {
		return nil, ErrInvalidState
	}
	product, err = addChecked(product, uint256.NewInt(5000))
	if err != nil {
		return nil, err
	}
	treasuryAssets := product.Div(product, uint256.NewInt(10000))
	return v3Utils.MulDiv(treasuryAssets, ray, rate)
}
