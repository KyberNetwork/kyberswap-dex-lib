package upshiftusdf

import "errors"

const (
	DexType = "upshift-usdf"

	// Vault is the Upshift (Fractal) USDF vault and the pool address; it pulls USDC and mints USDF.
	Vault = "0x3eaa4b3e8967c02ce1304c1eb35e8c5409838dfc"
	// USDF is the vault's ReceiptToken.
	USDF = "0x51acb1ea45c1ec2512ae4202b9076c13016dc8aa"
	USDC = "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"

	// Fork gasUsed - 21000 of deposit() to a recipient already holding USDF (the executor keeps dust;
	// a zero-balance recipient costs 17100 more): compute() writes fresh _records slots
	// per rolled day, so gas is gasNoRoll at 0 days, else gasRollBase + days*gasPerDay
	// (exact for 1-29 days, 104 over the measured 2820500 at 30).
	gasNoRoll   int64 = 76404
	gasRollBase int64 = 78904
	gasPerDay   int64 = 91390
)

const (
	secondsPerDay = 86400
	// maxRollDays is compute()'s per-call cap (`if (x >= 30) break`).
	maxRollDays = 30
	// Seconds around now in which a day boundary makes the quote unsafe.
	boundaryAfterSec  = 120 // tx lands after the quote
	boundaryBeforeSec = 30  // quote clock ahead of the chain clock
)

const (
	vaultMethodStartOfYear = "startOfYearTimestamp"
	vaultMethodPeriod      = "currentPeriod"
	vaultMethodMinDeposit  = "minDepositAmount"
	vaultMethodSnapshot    = "getSnapshot"
	tokenMethodMaxSupply   = "maxSupply"
	tokenMethodTotalSupply = "totalSupply"
)

var (
	ErrInvalidSwapPath  = errors.New("upshift-usdf: only USDC -> USDF is supported")
	ErrZeroAmount       = errors.New("upshift-usdf: zero amount")
	ErrOverflow         = errors.New("upshift-usdf: uint256 arithmetic overflow")
	ErrBelowMinDeposit  = errors.New("upshift-usdf: minimum deposit amount not met")
	ErrZeroPrice        = errors.New("upshift-usdf: token price is zero")
	ErrZeroShares       = errors.New("upshift-usdf: zero USDF minted")
	ErrSupplyExceeded   = errors.New("upshift-usdf: token supply limit exceeded")
	ErrPeriodBoundary   = errors.New("upshift-usdf: too close to a day boundary, roll is uncertain")
	ErrClockBeforeStart = errors.New("upshift-usdf: clock before startOfYearTimestamp")
	ErrInvalidExtra     = errors.New("upshift-usdf: invalid pool extra")
)
