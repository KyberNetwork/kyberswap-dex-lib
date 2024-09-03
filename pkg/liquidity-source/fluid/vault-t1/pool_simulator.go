package vaultT1

// import (
// 	"errors"
// 	"math/big"

// 	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
// 	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
// 	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
// 	"github.com/samber/lo"
// )

// var (
// 	ErrInvalidAmountIn = errors.New("invalid amountIn")
// )

// type PoolSimulator struct {
// 	poolpkg.Pool
// }

// var (
// 	defaultGas = Gas{Liquidate: 250000}
// )

// func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
// 	return &PoolSimulator{
// 		Pool: poolpkg.Pool{Info: poolpkg.PoolInfo{
// 			Address:     entityPool.Address,
// 			ReserveUsd:  entityPool.ReserveUsd,
// 			Exchange:    entityPool.Exchange,
// 			Type:        entityPool.Type,
// 			Tokens:      lo.Map(entityPool.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
// 			Reserves:    lo.Map(entityPool.Reserves, func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
// 			BlockNumber: entityPool.BlockNumber,
// 			SwapFee:     big.NewInt(0), // no swap fee on liquidations
// 		}},
// 	}, nil
// }

// func (s *PoolSimulator) CalcAmountOut(param poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
// 	if param.TokenAmountIn.Amount.Cmp(bignumber.ZeroBI) <= 0 {
// 		return nil, ErrInvalidAmountIn
// 	}

// 	// here we would need to do a call to the VaultLiquidationResolver exact Input method, which returns the amount out
// 	// that we can return further below for TokenAmountOut. docs for it:

// 	/// @notice finds all swaps from `tokenIn_` to `tokenOut_` for an exact input amount `inAmt_`.
// 	///         filters the available swaps and sorts them by ratio, so the returned swaps are the best available
// 	///         swaps to reach the target `inAmt_`.
// 	///         If the full available amount is less than the target `inAmt_`, the available amount is returned as `actualInAmt_`.
// 	/// @dev The only cases that are currently not best possible optimized for are when the ratio for withoutAbsorb is better
// 	/// but the target swap amount is more than the available without absorb liquidity. For this, currently the available
// 	/// withAbsorb liquidity is consumed first before tapping into the better ratio withoutAbsorb liquidity.
// 	/// The optimized version would be to split the tx into two swaps, first executing liquidate() with absorb = false
// 	/// to fully swap all the withoutAbsorb liquidity, and then in the second tx run with absorb = true to fill the missing
// 	/// amount up to the target amount with the worse ratio with absorb liquidity.
// 	/// @param tokenIn_ input token
// 	/// @param tokenOut_ output token
// 	/// @param inAmt_ exact input token amount that should be swapped to output token
// 	/// @return swaps_ swaps to reach the target amount, sorted by ratio in descending order
// 	///         (higher ratio = better rate). Best ratio swap will be at pos 0, second best at pos 1 and so on.
// 	/// @return actualInAmt_ actual input token amount. Can be less than inAmt_ if all available swaps can not cover
// 	///                      the target amount.
// 	/// @return outAmt_ output token amount received for `actualInAmt_`
// 	// function exactInput(
// 	//     address tokenIn_,
// 	//     address tokenOut_,
// 	//     uint256 inAmt_
// 	// ) public returns (Swap[] memory swaps_, uint256 actualInAmt_, uint256 outAmt_) {

// 	// req := u.ethrpcClient.R().SetContext(ctx)

// 	// req.AddCall(&ethrpc.Call{
// 	// 	ABI:    vaultLiquidationResolverABI,
// 	// 	Target: vaultLiquidationResolver[u.config.ChainID],
// 	// 	Method: VLRMethodGetAllSwapPaths,
// 	// }, []interface{}{&paths})

// 	// if _, err := req.Aggregate(); err != nil {
// 	// 	logger.WithFields(logger.Fields{
// 	// 		"dexType": DexType,
// 	// 	}).Error("aggregate request failed")
// 	// 	return nil, err
// 	// }

// 	return &poolpkg.CalcAmountOutResult{
// 		TokenAmountOut: &poolpkg.TokenAmount{Token: param.TokenOut, Amount: s.amountForShare(param.TokenAmountIn.Amount)},
// 		Fee:            &poolpkg.TokenAmount{Token: param.TokenOut, Amount: bignumber.ZeroBI},
// 		Gas:            defaultGas.Liquidate,
// 	}, nil
// }

// func (s *PoolSimulator) UpdateBalance(_ poolpkg.UpdateBalanceParams) {}

// func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
// 	return PoolMeta{
// 		BlockNumber: s.Pool.Info.BlockNumber,
// 	}
// }
