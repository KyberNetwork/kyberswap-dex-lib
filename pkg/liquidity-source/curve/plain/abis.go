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
)

func init() {
	var build = []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&curvePlainABI, curvePlainABIBytes},
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
