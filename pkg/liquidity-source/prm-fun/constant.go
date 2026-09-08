package prmfun

import "errors"

const (
	DexType = "prm-fun"

	// getReserves returns (quoteReserve_, tokenReserve_); virtualMeme()/virtualDesk() are
	// not in the vendor's published ABI.
	memeCurveMethodGetReserves = "getReserves"
	memeCurveMethodPhase       = "phase"
	memeCurveMethodMemeSold    = "memeSold"
	memeCurveMethodDeskRaised  = "deskRaised"

	memeFactoryMethodMemeCount  = "memeCount"
	memeFactoryMethodMemeTokens = "memeTokens"
	memeFactoryMethodGetMeme    = "getMeme"
)

// MemeCurve.Phase. Anything other than Trading is unswappable here: graduated pools move
// to the uniswap-v4-premium pool type, paused ones stop quoting.
const (
	PhaseTrading   uint8 = 0
	PhaseGraduated uint8 = 1
)

const (
	// MemeCurve's BPS/FEE_BPS, fixed protocol-wide rather than read per pool.
	bps         = 10_000
	feeBpsConst = 100

	// MemeCurve.SALE_SUPPLY: meme tokens sellable through the curve before graduation.
	saleSupply = "800000000000000000000000000"

	buyGas  = 150000
	sellGas = 130000
)

var (
	ErrPoolNotTrading        = errors.New("prm-fun: meme curve is not in Trading phase")
	ErrZeroAmount            = errors.New("prm-fun: zero amount")
	ErrInsufficientLiquidity = errors.New("prm-fun: insufficient curve liquidity")
	ErrInvalidToken          = errors.New("prm-fun: invalid token for this pool")
)
