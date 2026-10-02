package uscore

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/samber/lo"
)

var poolABI = lo.Must(abi.JSON(strings.NewReader(`[
  {"type":"function","name":"quote","stateMutability":"view","inputs":[],"outputs":[{"type":"address"}]},
  {"type":"function","name":"base","stateMutability":"view","inputs":[],"outputs":[{"type":"address"}]},
  {"type":"function","name":"getReserves","stateMutability":"view","inputs":[],"outputs":[{"name":"reserveQuote","type":"uint256"},{"name":"reserveBase","type":"uint256"}]},
  {"type":"function","name":"quoteExactIn","stateMutability":"view","inputs":[{"name":"tokenIn","type":"address"},{"name":"amountIn","type":"uint256"}],"outputs":[{"name":"amountOut","type":"uint256"},{"name":"fee","type":"uint256"},{"name":"status","type":"uint8"}]},
  {"type":"function","name":"quoteLadder","stateMutability":"view","inputs":[{"name":"tokenIn","type":"address"},{"name":"amountsIn","type":"uint256[]"}],"outputs":[{"name":"outs","type":"uint256[]"},{"name":"fees","type":"uint256[]"},{"name":"status","type":"uint8"}]}
]`)))

var clockABI = lo.Must(abi.JSON(strings.NewReader(`[
  {"type":"function","name":"getCurrentBlockTimestamp","stateMutability":"view","inputs":[],"outputs":[{"type":"uint256"}]}
]`)))
