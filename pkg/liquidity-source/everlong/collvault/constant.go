package everlongcollvault

import (
	"errors"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = "everlong-collvault"

	rebalancerMethodCollVault         = "collVault"
	rebalancerMethodSettlementSwapper = "settlementSwapper"
	rebalancerMethodExchangeState     = "exchangeState"
	swapperMethodAlm                  = "alm"
	almMethodGetTotalAmounts          = "getTotalAmounts"
	almMethodGetReservesAtReference   = "getReservesAtReference"
	erc20MethodTotalSupply            = "totalSupply"
	cvMethodTotalAssets               = "totalAssets"
	cvMethodGetWithdrawFee            = "getWithdrawFee"
	cvMethodAssetDecimals             = "assetDecimals"

	// defaultGasSwap: flash-mint, CollVault mint/redeem, ALM leg settlement and the CDP
	// adjust all run inside one swap. Estimate pending a live fill on the current stack
	// (the ALM's own deposit/withdraw is closed-form); overridable via Config.GasSwap.
	defaultGasSwap = 3_000_000

	// leverageBufferBps shaves the physical-CR-floor cap: the cap is computed from the
	// PRE-fill floor prediction, while the contract re-checks POST-fill (FillValueDrop)
	// along with an oracle-priced ICR floor and a price band, which can bind slightly
	// earlier at the exact boundary.
	leverageBufferBps = 200
)

var (
	ErrInvalidToken        = errors.New("invalid token")
	ErrInvalidAmountIn     = errors.New("invalid amount in")
	ErrNotPriceable        = errors.New("vault state is not locally priceable (degenerate region)")
	ErrZeroAmountOut       = errors.New("zero amount out")
	ErrSwapRejected        = errors.New("the vault rejects this fill")
	ErrNoCurveParams       = errors.New("no curve params for this chain — the deployed rebalancer's constants are required")
	ErrInvalidCurveParams  = errors.New("invalid curve params")
	ErrInvalidSnapshotWord = errors.New("invalid snapshot value")
)

// berachainCurveParams are the constants frozen into the deployed Berachain
// CollateralRebalancer + linked CollRebalancerMath (the 155%-wall "champion-v3"
// construction; read from the deployment source, parity-validated against the shipped
// library bytecode). Per-deployment values — a contract upgrade or another chain's
// deployment can differ, and Config.CurveParams overrides these.
func berachainCurveParams() CurveParams {
	wad := func(s string) *big.Int {
		v, _ := new(big.Int).SetString(s, 10)
		return v
	}
	return CurveParams{
		LeverageRatioWad: wad("444444444444444444"),
		HZero:            wad("562500000000000000"),
		HJoin:            wad("1010000000000000000"),
		HWall:            wad("1882448291726770582"),
		Width:            wad("872448291726770582"),
		DJoin:            wad("509975124224178054"),
		DWall:            wad("1214482768855981020"),
		RescueSpreadPpm:  big.NewInt(13_000),
		BezierPhi: [4]*big.Int{
			wad("995037190209989135"), wad("851783312849706840"),
			wad("738044106433170508"), wad("645161290322580645"),
		},
		BezierIntegral: [5]*big.Int{
			big.NewInt(0), wad("248759297552497283"), wad("461705125764923993"),
			wad("646216152373216620"), wad("807506474953861782"),
		},
		PhysicalCrFloorWad: wad("1820000000000000000"),
	}
}

// curveParamsByChain carries the per-deployment curve constants; Config.CurveParams
// overrides them, and chains without an entry must supply the override.
var curveParamsByChain = map[valueobject.ChainID]func() CurveParams{
	valueobject.ChainIDBerachain: berachainCurveParams,
}
