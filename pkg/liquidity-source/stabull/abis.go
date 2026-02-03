package stabull

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	stabullPoolABI         abi.ABI
	assimilatorABI         abi.ABI
	chainlinkAggregatorABI abi.ABI
)

func init() {
	var err error

	// Parse Pool ABI
	stabullPoolABI, err = abi.JSON(bytes.NewReader(stabullPoolABIData))
	if err != nil {
		panic(err)
	}

	// Parse Assimilator ABI
	assimilatorABI, err = abi.JSON(bytes.NewReader(assimilatorABIData))
	if err != nil {
		panic(err)
	}

	// Parse Chainlink Aggregator ABI
	chainlinkAggregatorABI, err = abi.JSON(bytes.NewReader(chainlinkAggregatorABIData))
	if err != nil {
		panic(err)
	}
}
