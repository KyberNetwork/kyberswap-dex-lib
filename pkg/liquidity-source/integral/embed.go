package integral

import _ "embed"

//go:embed abis/Reserves.json
var twapReservesJSON []byte

//go:embed abis/TwapFactory.json
var twapFactoryJSON []byte

//go:embed abis/TwapPair.json
var twapPairJSON []byte

//go:embed abis/ITwapOracle.json
var twapOracleJSON []byte
