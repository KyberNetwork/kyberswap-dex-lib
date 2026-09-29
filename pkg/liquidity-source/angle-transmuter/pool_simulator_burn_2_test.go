package angletransmuter

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func Test_ReadBurn_scUSD(t *testing.T) {
	// txhash sonic: 0x6ac61aca988adfddc56f7a792ae39597c7e25c152f6b0a3d60ecbe2d0507afe5
	p := getParallelPool()
	expectedValue := setUInt("998601150000000000")
	targetPrice, err := p._read(STABLE, p.Transmuter.Collaterals["0xd3DCe716f3eF535C5Ff8d041c1A41C3bd89b97aE"].Config.TargetFeed, BASE_18)
	assert.Nil(t, err)
	assert.Equal(t, setUInt("1000000000000000000"), targetPrice)

	oraclePrice, err := p._read(CHAINLINK_FEEDS, p.Transmuter.Collaterals["0xd3DCe716f3eF535C5Ff8d041c1A41C3bd89b97aE"].Config.OracleFeed, targetPrice)
	assert.Nil(t, err)
	assert.Equal(t, expectedValue, oraclePrice)

	// adjust based on UserDeviation
	spot, target, err := p._readSpotAndTarget("0xd3DCe716f3eF535C5Ff8d041c1A41C3bd89b97aE")
	assert.Nil(t, err)
	assert.Equal(t, BASE_18, target)
	assert.Equal(t, expectedValue, spot)

	// adjust based on BurnRatioDeviation
	oracleValue, ratio, err := p._readBurn("0xd3DCe716f3eF535C5Ff8d041c1A41C3bd89b97aE")
	assert.Nil(t, err)
	assert.Equal(t, oracleValue, spot)
	assert.Equal(t, expectedValue, ratio)
}

func Test_ReadBurn_ygami_scUSD(t *testing.T) {
	// txhash sonic: 0x95922b141227ccc542727f9a1501c91dface2c14bd3d52b63807ef25266c0a59
	p := getParallelPool()

	targetPrice, err := p._read(MAX, p.Transmuter.Collaterals["0xA19ebd8f9114519bF947671021c01d152c3777E4"].Config.TargetFeed, BASE_18)
	assert.Nil(t, err)
	assert.Equal(t, setUInt("998767916392050000"), targetPrice)

	oraclePrice, err := p._read(MORPHO_ORACLE, p.Transmuter.Collaterals["0xA19ebd8f9114519bF947671021c01d152c3777E4"].Config.OracleFeed, targetPrice)
	assert.Nil(t, err)
	assert.Equal(t, setUInt("998775905201250000"), oraclePrice)

	// adjust based on UserDeviation
	spot, target, err := p._readSpotAndTarget("0xA19ebd8f9114519bF947671021c01d152c3777E4")
	assert.Nil(t, err)
	assert.Equal(t, targetPrice, target)
	assert.Equal(t, oraclePrice, spot)

	// adjust based on BurnRatioDeviation
	oracleValue, ratio, err := p._readBurn("0xA19ebd8f9114519bF947671021c01d152c3777E4")
	assert.Nil(t, err)
	assert.Equal(t, oracleValue, spot)
	assert.Equal(t, BASE_18, ratio)
}

func Test_GetBurnOracle_scUSD(t *testing.T) {
	// txhash sonic: 0x6ac61aca988adfddc56f7a792ae39597c7e25c152f6b0a3d60ecbe2d0507afe5
	p := getParallelPool()
	oracleValue, minRatio, err := p._getBurnOracle("0xd3DCe716f3eF535C5Ff8d041c1A41C3bd89b97aE")
	assert.Nil(t, err)
	assert.Equal(t, setUInt("998601150000000000"), oracleValue)
	assert.Equal(t, setUInt("998601150000000000"), minRatio)
}

func Test_GetBurnOracle_ygami_scUSD(t *testing.T) {
	// txhash sonic: 0x95922b141227ccc542727f9a1501c91dface2c14bd3d52b63807ef25266c0a59
	p := getParallelPool()
	oracleValue, minRatio, err := p._getBurnOracle("0xA19ebd8f9114519bF947671021c01d152c3777E4")
	assert.Nil(t, err)
	assert.Equal(t, setUInt("998775905201250000"), oracleValue)
	assert.Equal(t, setUInt("998601150000000000"), minRatio)
}

func Test_quoteBurnExactInput_scUSD(t *testing.T) {
	// txhash sonic: 0x6ac61aca988adfddc56f7a792ae39597c7e25c152f6b0a3d60ecbe2d0507afe5
	p := getParallelPool()
	amountIn := setUInt("13600000000000000")
	oracleValue, minRatio, err := p._getBurnOracle("0xd3DCe716f3eF535C5Ff8d041c1A41C3bd89b97aE")
	assert.Nil(t, err)
	assert.Equal(t, setUInt("998601150000000000"), oracleValue)
	assert.Equal(t, setUInt("998601150000000000"), minRatio)
	collatInfo := p.Transmuter.Collaterals["0xd3DCe716f3eF535C5Ff8d041c1A41C3bd89b97aE"]

	amountOutAfterFee, err := _quoteFees(
		&collatInfo,
		BurnExactInput,
		amountIn,
		new(uint256.Int).Sub(p.Transmuter.TotalStablecoinIssued, p.Transmuter.Collaterals["0xd3DCe716f3eF535C5Ff8d041c1A41C3bd89b97aE"].StablecoinsIssued),
		p.Transmuter.TotalStablecoinIssued,
	)
	assert.Nil(t, err)
	assert.Equal(t, amountIn, amountOutAfterFee)

	amountOut, err := _quoteBurnExactInput(
		oracleValue, minRatio, amountIn,
		&collatInfo,
		new(uint256.Int).Sub(p.Transmuter.TotalStablecoinIssued, p.Transmuter.Collaterals["0xd3DCe716f3eF535C5Ff8d041c1A41C3bd89b97aE"].StablecoinsIssued),
		6,
		p.Transmuter.TotalStablecoinIssued,
	)
	assert.Nil(t, err)
	assert.Equal(t, setUInt("13600"), amountOut)
}

func Test_quoteBurnExactInput_ygami_scUSD(t *testing.T) {
	// txhash sonic: 0x95922b141227ccc542727f9a1501c91dface2c14bd3d52b63807ef25266c0a59
	p := getParallelPool()
	amountIn := setUInt("27400000000000000")
	oracleValue, minRatio, err := p._getBurnOracle("0xA19ebd8f9114519bF947671021c01d152c3777E4")
	assert.Nil(t, err)
	assert.Equal(t, setUInt("998775905201250000"), oracleValue)
	assert.Equal(t, setUInt("998601150000000000"), minRatio)
	collatInfo := p.Transmuter.Collaterals["0xA19ebd8f9114519bF947671021c01d152c3777E4"]

	amountOutAfterFee, err := _quoteFees(
		&collatInfo,
		BurnExactInput,
		amountIn,
		new(uint256.Int).Sub(p.Transmuter.TotalStablecoinIssued, p.Transmuter.Collaterals["0xA19ebd8f9114519bF947671021c01d152c3777E4"].StablecoinsIssued),
		p.Transmuter.TotalStablecoinIssued,
	)
	assert.Nil(t, err)
	assert.Equal(t, setUInt("27386300000000000"), amountOutAfterFee)

	amountOut, err := _quoteBurnExactInput(
		oracleValue, minRatio, amountIn,
		&collatInfo,
		new(uint256.Int).Sub(p.Transmuter.TotalStablecoinIssued, p.Transmuter.Collaterals["0xA19ebd8f9114519bF947671021c01d152c3777E4"].StablecoinsIssued),
		6,
		p.Transmuter.TotalStablecoinIssued,
	)
	assert.Nil(t, err)
	assert.Equal(t, setUInt("27381"), amountOut)
}

func TestCalcAmountOut_scUSD(t *testing.T) {
	p := getParallelPool()
	res, err := p.CalcAmountOut(
		pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0x08417cdb7F52a5021bB4eb6E0deAf3f295c3f182",
				Amount: setUInt("3390079323519859415728").ToBig(),
			},
			TokenOut: "0xd3DCe716f3eF535C5Ff8d041c1A41C3bd89b97aE",
		},
	)
	assert.Nil(t, err)
	assert.Equal(t, setUInt("3390079323").ToBig(), res.TokenAmountOut.Amount)
}

func TestCalcAmountOut_ygami_scUSD(t *testing.T) {
	p := getParallelPool()
	res, err := p.CalcAmountOut(
		pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0x08417cdb7F52a5021bB4eb6E0deAf3f295c3f182",
				Amount: setUInt("3390079323519859415728").ToBig(),
			},
			TokenOut: "0xA19ebd8f9114519bF947671021c01d152c3777E4",
		},
	)
	assert.Nil(t, err)
	assert.Equal(t, setUInt("3387791420").ToBig(), res.TokenAmountOut.Amount)
}

// Regression test for the Plasma/Arbitrum-style rail with zero mint history.
// Arbitrum USDA transmuter 0xd253b62108d1831aed298fc2434a5a8e4e418053 holds USDC as its
// only collateral with getIssuedByCollateral(USDC) = 0; every burn size reverts on-chain
// in Swapper._swap with an arithmetic underflow on normalizedStables
// (see AngleProtocol/angle-transmuter Swapper.sol, burn branch comment).
func TestCalcAmountOut_burn_zero_issuance_rejected(t *testing.T) {
	const (
		usdc = "0xaf88d065e77c8cc2239327c5edb3a432268e5831"
		usda = "0x0000206329b97db379d5e1bf586bbdb969c63274"
	)
	p := PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Tokens: []string{usdc, usda},
		}},
		Decimals: []uint8{6, 18},
		Transmuter: TransmuterState{
			// Values mirror the router-api snapshot of pool 0xd253b62108d1831aed298fc2434a5a8e4e418053
			// (router-api.kyberengineering.io/arbitrum/api/v1/pools?ids=<pool>): totalStablecoinIssued=0,
			// USDC collateral stablecoinsIssued=0, normalizedStables=0, balance=2215112.
			TotalStablecoinIssued: uint256.NewInt(0),
			Collaterals: map[string]CollateralState{
				usdc: {
					IsBurnLive:        true,
					IsMintLive:        true,
					StablecoinsIssued: uint256.NewInt(0),
					NormalizedStables: uint256.NewInt(0),
					Balance:           setUInt("2215112"),
					Fees: Fees{
						XFeeBurn: []*uint256.Int{uint256.NewInt(0)},
						YFeeBurn: []*uint256.Int{uint256.NewInt(0)},
					},
					Config: Oracle{
						OracleType: STABLE,
						TargetType: STABLE,
						Hyperparameters: Hyperparameters{
							UserDeviation:      uint256.NewInt(0),
							BurnRatioDeviation: uint256.NewInt(0),
						},
					},
				},
			},
		},
	}

	amountIn := setUInt("1372531263530234068")

	collatInfo := p.Transmuter.Collaterals[usdc]
	amountOut, err := _quoteBurnExactInput(
		newBASE18(), newBASE18(), amountIn,
		&collatInfo,
		uint256.NewInt(0),
		p.Decimals[0],
		uint256.NewInt(0),
	)
	assert.Nil(t, err)
	assert.Equal(t, setUInt("1372531"), amountOut)

	res, err := p.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: usda, Amount: amountIn.ToBig()},
		TokenOut:      usdc,
	})
	assert.Nil(t, res)
	assert.ErrorIs(t, err, ErrInsufficientStablecoinIssued)
}
