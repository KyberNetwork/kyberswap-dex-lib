package stonkbrokersfunv2

import _ "embed"

//go:embed abi/Pad.json
var padBytes []byte

//go:embed abi/Lens.json
var lensBytes []byte

// aggregatorV3Bytes / twapPoolBytes are minimal, hand-written ABIs for the
// two third-party contracts read for the buy-side oracle gate: a standard
// Chainlink AggregatorV3Interface (decimals, latestRoundData) and a minimal
// Uniswap-v3-style pool (token0, token1, observe). Selectors verified via
// `cast sig` against the well-known canonical signatures -- see
// context/stonkbrokers/output/math.md.
//
//go:embed abi/AggregatorV3.json
var aggregatorV3Bytes []byte

//go:embed abi/TwapPool.json
var twapPoolBytes []byte
