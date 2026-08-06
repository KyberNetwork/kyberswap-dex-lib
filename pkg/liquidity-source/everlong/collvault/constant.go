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

	// defaultGasSwap is from a live Katana fill through the settlement pipeline
	// (deleverage tx 0xb34ac182…, 5,889,694 gas): flash-mint, CollVault mint/redeem, CL
	// pool add/remove liquidity and the CDP adjust all run inside one swap.
	defaultGasSwap = 6_000_000

	// leverageBufferBps shaves the physical-CR-floor cap: the cap is computed from the
	// PRE-fill floor prediction, while the contract re-checks POST-fill (FillValueDrop),
	// which can bind slightly earlier at the exact boundary.
	leverageBufferBps = 200
)

var (
	ErrInvalidToken        = errors.New("invalid token")
	ErrInvalidAmountIn     = errors.New("invalid amount in")
	ErrNotPriceable        = errors.New("vault state is not locally priceable (recovery/degenerate region)")
	ErrZeroAmountOut       = errors.New("zero amount out")
	ErrSwapRejected        = errors.New("the vault rejects this fill")
	ErrNoCurveParams       = errors.New("no curve params for this chain — the deployed rebalancer's constants are required")
	ErrInvalidCurveParams  = errors.New("invalid curve params")
	ErrInvalidSnapshotWord = errors.New("invalid snapshot value")
)

// katanaCurveParams are the constants frozen into the deployed Katana
// CollateralRebalancer (read from the deployed verified source; parity-validated
// wei-exact against the live quote oracle). Per-deployment values — a contract upgrade
// or another chain's deployment can differ.
func katanaCurveParams() CurveParams {
	wad := func(s string) *big.Int {
		v, _ := new(big.Int).SetString(s, 10)
		return v
	}
	return CurveParams{
		LeverageRatioWad: wad("444444444444444444"),
		HZero:            wad("562500000000000000"),
		HJoin:            wad("1010000000000000000"),
		HWall:            wad("1281238087385139984"),
		Width:            wad("271238087385139984"),
		DJoin:            wad("509975124224178054"),
		DWall:            wad("719796678306258417"),
		RescueSpreadPpm:  big.NewInt(13_000),
		BezierPhi: [4]*big.Int{
			wad("995037190209989135"), wad("950500559129190259"),
			wad("586943123760998626"), wad("561797752808988764"),
		},
		BezierIntegral: [5]*big.Int{
			big.NewInt(0), wad("248759297552497283"), wad("486384437334794848"),
			wad("633120218275044505"), wad("773569656477291696"),
		},
		PhysicalCrFloorWad: wad("1840000000000000000"),
	}
}

// curveParamsByChain carries the per-deployment curve constants. Berachain is added
// once its deployment's constants are confirmed from the deployed source.
var curveParamsByChain = map[valueobject.ChainID]func() CurveParams{
	valueobject.ChainIDKatana: katanaCurveParams,
}
