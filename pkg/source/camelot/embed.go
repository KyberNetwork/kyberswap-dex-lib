package camelot

import _ "embed"

//go:embed abis/CamelotFactory.json
var camelotFactoryBytes []byte

//go:embed abis/CamelotPair.json
var camelotPairBytes []byte
