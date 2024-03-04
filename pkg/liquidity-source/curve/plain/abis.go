package plain

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/shared"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	curvePlainABI abi.ABI

	// some ABIs depend on number of tokens, so we create one for each of them
	numTokenDependedABIs [shared.MaxTokenCount]abi.ABI

	numTokenDependedABITemplate = `[
		{"stateMutability":"view","type":"function","name":"stored_rates","inputs":[],"outputs":[{"name":"","type":"uint256[{NUM_TOKEN}]"}]}
	]`

	// some old pools use int128 input instead of uint256
	getBalances128ABI      abi.ABI
	getBalances128ABIBytes = []byte(`[
		{ "name": "balances", "outputs": [{ "type": "uint256", "name": "" }], "inputs": [{ "type": "int128", "name": "arg0" }], "constant": true, "payable": false, "type": "function"}
	]`)
)

func init() {
	var build = []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&curvePlainABI, curvePlainABIBytes},
		{&getBalances128ABI, getBalances128ABIBytes},
	}

	var err error
	for _, b := range build {
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}

	for i := range numTokenDependedABIs {
		numTokenDependedABIJson := strings.ReplaceAll(numTokenDependedABITemplate, "{NUM_TOKEN}", strconv.Itoa(i))
		numTokenDependedABIs[i], err = abi.JSON(strings.NewReader(numTokenDependedABIJson))
		if err != nil {
			panic(err)
		}
	}
}
