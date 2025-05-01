package hyeth

import (
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
)

func (s *PoolSimulator) deposit(asset *uint256.Int) (*uint256.Int, error) {
	if asset.Gt(s.maxDeposit) {
		return nil, ErrERC4626DepositMoreThanMax
	}

	return s.previewDeposit(asset)
}

func (s *PoolSimulator) previewDeposit(asset *uint256.Int) (*uint256.Int, error) {
	shares, overflow := new(uint256.Int).MulDivOverflow(asset, s.componentTotalSupply, s.componentTotalAsset)
	if overflow {
		return nil, number.ErrOverflow
	}

	return shares, nil
}

func (s *PoolSimulator) redeem(shares *uint256.Int) (*uint256.Int, error) {
	if shares.Gt(s.maxRedeem) {
		return nil, ErrERC4626RedeemMoreThanMax
	}

	return s.previewRedeem(shares)
}

func (s *PoolSimulator) previewRedeem(shares *uint256.Int) (*uint256.Int, error) {
	assets, overflow := new(uint256.Int).MulDivOverflow(
		shares,
		new(uint256.Int).Add(s.componentTotalAsset, uint256.NewInt(1)),
		new(uint256.Int).Add(s.componentTotalSupply, uint256.NewInt(1)),
	)
	if overflow {
		return nil, number.ErrOverflow
	}

	return assets, nil
}

/*
	https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/develop/src/contracts/executor-helpers/ExecutorHelper9.sol#L473
	CalcAmountOut before calling issueExactSetFromETH
*/

func (s *PoolSimulator) issueSetFromETH(_quantity *uint256.Int) (*uint256.Int, error) {
	componentShares, err := s.deposit(_quantity)
	if err != nil {
		return nil, err
	}
	return s.getRequiredAmountSetToken(componentShares), nil
}

func (s *PoolSimulator) getRequiredAmountSetToken(componentShares *uint256.Int) *uint256.Int {
	equityUnit := new(uint256.Int).Set(s._getTotalIssuanceUnitsFromBalances())
	// uint256 _quantityAfterFee = (componentShares * 10 ** 18) / uint256(cumulativeEquity);
	quantityAfterFee := new(uint256.Int).Div(
		new(uint256.Int).Mul(componentShares, U_1e18),
		equityUnit,
	)

	// tokenAmountOut = (_quantityAfterFee * 10 ** 18) / (10 ** 18 + swapData.issueFeeRate);
	tokenAmountOut := new(uint256.Int).Div(
		new(uint256.Int).Mul(quantityAfterFee, U_1e18),
		new(uint256.Int).Add(U_1e18, s.managerIssueFee),
	)
	return tokenAmountOut
}

/**
 * Redeem exact amount of SetToken for ETH ()
 * FlashMintHyETHV3.sol#L227 - https://etherscan.io/address/0xCb1eEA349f25288627f008C5e2a69b684bddDf49#code
 *
 * @param _amountSetToken   Amount of SetToken to redeem
 */
func (s *PoolSimulator) redeemSetForETH(amountSetToken *uint256.Int) (*uint256.Int, error) {
	position := s.getRequiredComponentRedemptionUnits(amountSetToken)
	// FlashMintHyETHV3.sol#L512
	assetAmount, err := s.redeem(position)
	if err != nil {
		return nil, err
	}
	return s._swapExactTokenForEth(s.component, assetAmount), nil
}

/**
 * Calculates the amount of each component will be returned on redemption
 * DebtIssuanceModule.sol#L414 - https://etherscan.io/address/0x04b59F9F09750C044D7CfbC177561E409085f0f3#code
 *
 * @param _quantity         Amount of Sets to be redeemed
 */
func (s *PoolSimulator) getRequiredComponentRedemptionUnits(_quantity *uint256.Int) *uint256.Int {
	totalQuantity := s.calculateTotalFees(_quantity, false)
	return s._calculateRequiredComponentIssuanceUnits(totalQuantity, false)
}

/**
 * Calculates the manager fee, protocol fee and resulting totalQuantity to use when calculating unit amounts. If fees are charged they
 * are added to the total issue quantity, for example 1% fee on 100 Sets means 101 Sets are minted by caller, the _to address receives
 * 100 and the feeRecipient receives 1. Conversely, on redemption the redeemer will only receive the collateral that collateralizes 99
 * Sets, while the additional Set is given to the feeRecipient.
 * DebtIssuanceModule.sol#L356 - https://etherscan.io/address/0x04b59F9F09750C044D7CfbC177561E409085f0f3#code
 *
 * @param _quantity                 Amount of SetToken issuer wants to receive/redeem
 * @param _isIssue                  If issuing or redeeming
 */
func (s *PoolSimulator) calculateTotalFees(_quantity *uint256.Int, _isIssue bool) *uint256.Int {
	totalFeeRate := lo.Ternary(_isIssue, s.managerIssueFee, s.managerRedeemFee)
	totalFee := new(uint256.Int).Mul(totalFeeRate, _quantity)
	totalFee.Div(totalFee, U_1e18)

	totalQuantity := new(uint256.Int)
	if _isIssue {
		totalQuantity.Add(_quantity, totalFee)
	} else {
		totalQuantity.Sub(_quantity, totalFee)
	}
	return totalQuantity
}

/**
 * Calculates the amount of each component needed to collateralize passed issue quantity of Sets
 * DebtIssuanceModule.sol#L453 - https://etherscan.io/address/0x04b59F9F09750C044D7CfbC177561E409085f0f3#code
 *
 * @param _quantity         Amount of Sets to be redeemed
 * @param _isIssue          Whether Sets are being issued or redeemed (rounding process)
 */
func (s *PoolSimulator) _calculateRequiredComponentIssuanceUnits(_quantity *uint256.Int, _ bool) *uint256.Int {
	equityUnit := new(uint256.Int).Set(s._getTotalIssuanceUnits())
	totalEquityUnit := equityUnit.Mul(equityUnit, _quantity).Div(equityUnit, U_1e18)
	return totalEquityUnit
}

/**
* Sums total debt and equity units for each component, taking into account default and external positions.
* DebtIssuanceModule.sol#L495 - https://etherscan.io/address/0x04b59F9F09750C044D7CfbC177561E409085f0f3#code
*
 */
func (s *PoolSimulator) _getTotalIssuanceUnits() *uint256.Int {
	equityUnit := s.defaultPositionRealUnit
	for _, externalPositionRealUnit := range s.externalPositionRealUnits {
		if externalPositionRealUnit.Sign() > 0 {
			equityUnit.Add(equityUnit, externalPositionRealUnit)
		}
	}

	return equityUnit
}

func (s *PoolSimulator) _getTotalIssuanceUnitsFromBalances() *uint256.Int {
	// cumulativeEquity = (balance * 1e18) / totalSupply
	equityUnit := new(uint256.Int).Div(
		new(uint256.Int).Mul(s.componentHyethBalance, U_1e18),
		s.hyethTotalSupply,
	)

	for _, externalPositionRealUnit := range s.externalPositionRealUnits {
		if externalPositionRealUnit.Sign() > 0 {
			equityUnit.Add(equityUnit, externalPositionRealUnit)
		}
	}

	return equityUnit
}

/**
 * @dev Convert specified token to ETH, either swapping or simply withdrawing if inputToken is WETH
 * FlashMintHyETHV3.sol#L665 - https://etherscan.io/address/0xCb1eEA349f25288627f008C5e2a69b684bddDf49#code
 */
func (s *PoolSimulator) _swapExactTokenForEth(_ common.Address, _amountIn *uint256.Int) *uint256.Int {
	return _amountIn
}
