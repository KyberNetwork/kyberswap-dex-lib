package constant

import (
	"time"
)

/*
dex id - swap fee - for encode in simple swap
1 - 30 (default) = 0.3%
2 - 25
3 - 20
4 - 15
5 - 10
6 - 5
7 - 17 = 0.17%
8 - 18 = 0.18%
9 - 50 = 0.5%
*/
var (
	DexIds = map[string]uint16{
		"pancake":          2,
		"pancake-legacy":   2,
		"apeswap":          3,
		"wault":            3,
		"biswap":           5,
		"polydex":          5,
		"jetswap":          5,
		"polycat":          2,
		"spookyswap":       3,
		"axial":            3,
		"cronaswap":        2,
		"gravity":          2,
		"kyberswap":        0,
		"kyberswap-static": 0,
		"mmf":              7,
		"kryptodex":        3,
		"cometh":           9,
		"dinoswap":         8,
		"safeswap":         2,
		"pantherswap":      3,
		"morpheus":         4,
		"swapr":            2,
		"wagyuswap":        3,
		"astroswap":        3,
		"dystopia":         6,
	} // Default 1
	DexIdsByChain = map[int]map[string]uint16{
		BSCMAINNET: {
			"jetswap": 1,
		},
	}
	// for encode in simple swap and normal swap
	DexTypes = map[string]uint16{
		"curve":            2,
		"dmm":              3,
		"kyberswap":        3,
		"kyberswap-static": 3,
		"oneswap":          1,
		"ellipsis":         2,
		"nerve":            1,
		"iron-stable":      4,
		"balancer":         6,
		"synapse":          4,
		"saddle":           4,
		"axial":            4,
		"beethovenx":       6,
		"uniswapv3":        5,
		"kyberswapv2":      5,
		"dodo":             8,
		"velodrome":        9,
		"platypus":         10,
		"gmx":              11,
		"madmex":           11,
		"dystopia":         9,
		"synthetix":        12,
		"maker-psm":        13,
	}
	EtherAddress            string        = "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE"
	SwapMethodName          string        = "swap"
	SimpleSwapMethodName    string        = "swapSimpleMode"
	DefaultDeadlineInMinute time.Duration = time.Minute * 20
	ExactInput              string        = "EXACT_INPUT"
	ExactOutput             string        = "EXACT_OUTPUT"
	DefaultSlippage         int64         = 50
	MaximumSlippage         int64         = 2000
	MaxAmountInUSD          float64       = 100000000
	ChargeFeeByCurrencyIn   string        = "currency_in"
	ChargeFeeByCurrencyOut  string        = "currency_out"
	AddressZero                           = "0x0000000000000000000000000000000000000000"
)
